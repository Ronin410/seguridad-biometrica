package handlers

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"staffattendance/internal/rekognition"
)

// AsistenciaHandler implementa el marcaje de entrada/salida (SPEC.md
// sección 6.2, con el respaldo manual de la sección 8) y los reportes
// (sección 6.4).
type AsistenciaHandler struct {
	DB                  *sql.DB
	Rek                 *rekognition.Client
	Location            *time.Location
	SimilarityThreshold float32
}

func NewAsistenciaHandler(db *sql.DB, rek *rekognition.Client, location *time.Location, similarityThreshold float32) *AsistenciaHandler {
	return &AsistenciaHandler{DB: db, Rek: rek, Location: location, SimilarityThreshold: similarityThreshold}
}

func (h *AsistenciaHandler) Register(router gin.IRouter) {
	router.POST("/asistencia/marcar", h.marcar)
	router.POST("/asistencia/marcar-manual", h.marcarManual)
	router.GET("/asistencia/hoy", h.hoy)
	router.GET("/reportes/asistencia", h.reporte)
}

type turnoInfo struct {
	horaEntrada    string
	horaSalida     string
	toleranciaMin  int
	diasAplicables []int
}

type empleadoInfo struct {
	id     int
	nombre string
	turno  *turnoInfo
}

const marcajeDuplicadoVentana = 60 * time.Second

// marcar implementa el flujo principal de la sección 6.2: reconoce el
// rostro, decide automáticamente si es entrada o salida según el último
// registro del día, y compara contra el turno asignado para puntualidad.
func (h *AsistenciaHandler) marcar(c *gin.Context) {
	var input struct {
		Imagen string `json:"imagen" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Imagen requerida"})
		return
	}

	imageBytes, err := base64.StdEncoding.DecodeString(input.Imagen)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Imagen inválida"})
		return
	}

	negID := negocioID(c)
	faceID, similarity, err := h.Rek.SearchFace(c.Request.Context(), rekognition.CollectionID(negID), imageBytes, h.SimilarityThreshold)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rostro no reconocido"})
		return
	}

	empleado, err := h.buscarEmpleado("e.rekognition_face_id = $1 AND e.estado = 'activo' AND e.negocio_id = $2", faceID, negID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Empleado no encontrado o inactivo"})
		return
	}

	tipo, estado, horaReal, err := h.registrarMarcaje(empleado, negID, "facial", "", nil)
	if err != nil {
		respondMarcajeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"empleado_id": empleado.id,
		"nombre":      empleado.nombre,
		"tipo":        tipo,
		"estado":      estado,
		"hora":        horaReal.In(h.Location).Format("03:04 pm"),
		"similitud":   similarity,
		"mensaje":     mensajeBienvenida(empleado.nombre, tipo, horaReal.In(h.Location)),
	})
}

// marcarManual es el respaldo de la sección 8 cuando falla Rekognition (o,
// con capturado_en, cuando la PWA sincroniza un marcaje que se guardó
// localmente porque la tablet se quedó sin internet).
func (h *AsistenciaHandler) marcarManual(c *gin.Context) {
	var input struct {
		EmpleadoID  int    `json:"empleado_id" binding:"required"`
		Tipo        string `json:"tipo" binding:"required"`
		CapturadoEn string `json:"capturado_en"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || (input.Tipo != "entrada" && input.Tipo != "salida") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos, tipo debe ser 'entrada' o 'salida'"})
		return
	}

	var horaOverride *time.Time
	if input.CapturadoEn != "" {
		parsed, err := time.Parse(time.RFC3339, input.CapturadoEn)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "capturado_en debe ser una fecha ISO 8601 válida"})
			return
		}
		horaOverride = &parsed
	}

	negID := negocioID(c)
	empleado, err := h.buscarEmpleado("e.id = $1 AND e.negocio_id = $2", input.EmpleadoID, negID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Empleado no encontrado"})
		return
	}

	tipo, estado, horaReal, err := h.registrarMarcaje(empleado, negID, "manual", input.Tipo, horaOverride)
	if err != nil {
		respondMarcajeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"empleado_id": empleado.id,
		"nombre":      empleado.nombre,
		"tipo":        tipo,
		"estado":      estado,
		"hora":        horaReal.In(h.Location).Format("03:04 pm"),
	})
}

func (h *AsistenciaHandler) buscarEmpleado(whereClause string, args ...interface{}) (*empleadoInfo, error) {
	var e empleadoInfo
	var turnoID sql.NullInt64
	var horaEntrada, horaSalida sql.NullTime
	var toleranciaMin sql.NullInt64
	var diasAplicables pq.Int64Array

	query := fmt.Sprintf(`
		SELECT e.id, e.nombre_completo, t.id, t.hora_entrada, t.hora_salida, t.tolerancia_retardo_minutos, t.dias_aplicables
		FROM empleados e
		LEFT JOIN turnos t ON e.turno_id = t.id
		WHERE %s`, whereClause)

	err := h.DB.QueryRow(query, args...).Scan(
		&e.id, &e.nombre, &turnoID, &horaEntrada, &horaSalida, &toleranciaMin, &diasAplicables,
	)
	if err != nil {
		return nil, err
	}

	if turnoID.Valid {
		e.turno = &turnoInfo{
			horaEntrada:    horaEntrada.Time.Format("15:04:05"),
			horaSalida:     horaSalida.Time.Format("15:04:05"),
			toleranciaMin:  int(toleranciaMin.Int64),
			diasAplicables: toIntSlice(diasAplicables),
		}
	}

	return &e, nil
}

var errMarcajeDuplicado = errors.New("marcaje duplicado")

func respondMarcajeError(c *gin.Context, err error) {
	if errors.Is(err, errMarcajeDuplicado) {
		c.JSON(http.StatusConflict, gin.H{"error": "Marcaje duplicado, espera un momento antes de volver a intentar"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar la asistencia"})
}

// registrarMarcaje decide el tipo (si tipoForzado viene vacío), calcula la
// puntualidad contra el turno asignado y guarda el registro. horaOverride
// permite registrar con la hora real de captura en vez de "ahora" — lo usa
// la sincronización de marcajes guardados offline en la PWA (SPEC.md
// sección 8).
func (h *AsistenciaHandler) registrarMarcaje(empleado *empleadoInfo, negID int, metodo, tipoForzado string, horaOverride *time.Time) (tipo, estado string, horaReal time.Time, err error) {
	ahora := time.Now().In(h.Location)
	if horaOverride != nil {
		ahora = horaOverride.In(h.Location)
	}
	fecha := ahora.Format("2006-01-02")

	ultimoTipo, ultimaHora, existeUltimo, err := h.ultimoRegistroHoy(empleado.id, fecha)
	if err != nil {
		return "", "", time.Time{}, err
	}

	if tipoForzado != "" {
		tipo = tipoForzado
	} else if !existeUltimo {
		tipo = "entrada"
	} else if ultimoTipo == "entrada" {
		tipo = "salida"
	} else {
		tipo = "entrada"
	}

	if existeUltimo && ahora.Sub(ultimaHora) < marcajeDuplicadoVentana {
		return "", "", time.Time{}, errMarcajeDuplicado
	}

	if tipo == "entrada" {
		estado = clasificarEntrada(ahora, empleado.turno)
	} else {
		estado = clasificarSalida(ahora, empleado.turno)
	}

	var estadoValue interface{}
	if estado != "" {
		estadoValue = estado
	}

	_, err = h.DB.Exec(
		`INSERT INTO registros_asistencia (negocio_id, empleado_id, fecha, hora_real, tipo, estado, metodo)
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		negID, empleado.id, fecha, ahora, tipo, estadoValue, metodo,
	)
	if err != nil {
		return "", "", time.Time{}, err
	}

	return tipo, estado, ahora, nil
}

func (h *AsistenciaHandler) ultimoRegistroHoy(empleadoID int, fecha string) (tipo string, horaReal time.Time, existe bool, err error) {
	err = h.DB.QueryRow(
		`SELECT tipo, hora_real FROM registros_asistencia
         WHERE empleado_id = $1 AND fecha = $2
         ORDER BY hora_real DESC LIMIT 1`,
		empleadoID, fecha,
	).Scan(&tipo, &horaReal)

	if err == sql.ErrNoRows {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, err
	}
	return tipo, horaReal, true, nil
}

// clasificarEntrada compara la hora real contra hora_entrada + tolerancia.
// Empleados sin turno asignado no tienen puntualidad calculable.
func clasificarEntrada(horaReal time.Time, turno *turnoInfo) string {
	if turno == nil {
		return ""
	}
	limite := atClock(horaReal, turno.horaEntrada).Add(time.Duration(turno.toleranciaMin) * time.Minute)
	if horaReal.After(limite) {
		return "retardo"
	}
	return "puntual"
}

// clasificarSalida marca salida anticipada si es antes de hora_salida.
func clasificarSalida(horaReal time.Time, turno *turnoInfo) string {
	if turno == nil {
		return ""
	}
	programada := atClock(horaReal, turno.horaSalida)
	if horaReal.Before(programada) {
		return "salida_anticipada"
	}
	return "puntual"
}

// atClock construye un time.Time con la fecha de `base` y la hora de texto
// "HH:MM:SS" que devuelve Postgres para columnas TIME.
func atClock(base time.Time, horaTexto string) time.Time {
	partes := strings.Split(horaTexto, ":")
	hora, min := 0, 0
	if len(partes) >= 2 {
		hora, _ = strconv.Atoi(partes[0])
		min, _ = strconv.Atoi(partes[1])
	}
	return time.Date(base.Year(), base.Month(), base.Day(), hora, min, 0, 0, base.Location())
}

func mensajeBienvenida(nombre, tipo string, horaReal time.Time) string {
	accion := "Bienvenido"
	if tipo == "salida" {
		accion = "Hasta luego"
	}
	primerNombre := strings.SplitN(nombre, " ", 2)[0]
	return fmt.Sprintf("%s, %s — %s", accion, primerNombre, horaReal.Format("03:04 pm"))
}

// hoy da la vista en tiempo real de la sección 6.3: estado de cada empleado
// activo según su último registro del día.
func (h *AsistenciaHandler) hoy(c *gin.Context) {
	fecha := time.Now().In(h.Location).Format("2006-01-02")

	rows, err := h.DB.Query(`
		SELECT e.id, e.nombre_completo, e.puesto,
		       COALESCE(r.tipo, 'ausente') AS ultimo_tipo,
		       r.hora_real, r.estado
		FROM empleados e
		LEFT JOIN LATERAL (
			SELECT tipo, hora_real, estado
			FROM registros_asistencia
			WHERE empleado_id = e.id AND fecha = $1
			ORDER BY hora_real DESC LIMIT 1
		) r ON true
		WHERE e.estado = 'activo' AND e.negocio_id = $2
		ORDER BY e.nombre_completo ASC`, fecha, negocioID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar asistencia de hoy"})
		return
	}
	defer rows.Close()

	resultado := []gin.H{}
	for rows.Next() {
		var id int
		var nombre, puesto, ultimoTipo string
		var horaReal sql.NullTime
		var estado sql.NullString

		if err := rows.Scan(&id, &nombre, &puesto, &ultimoTipo, &horaReal, &estado); err != nil {
			continue
		}

		item := gin.H{
			"empleado_id":     id,
			"nombre":          nombre,
			"puesto":          puesto,
			"estatus":         ultimoTipo,
			"registro_estado": nullableString(estado),
		}
		if horaReal.Valid {
			item["hora"] = horaReal.Time.In(h.Location).Format("03:04 pm")
		}
		resultado = append(resultado, item)
	}

	c.JSON(http.StatusOK, resultado)
}

func nullableString(value sql.NullString) interface{} {
	if value.Valid {
		return value.String
	}
	return nil
}
