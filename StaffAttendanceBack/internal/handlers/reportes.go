package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type registroDelDia struct {
	tipo     string
	horaReal time.Time
	estado   string
}

type reporteEmpleado struct {
	id     int
	nombre string
	turno  *turnoInfo
}

// reporte implementa la sección 6.4: totales de días trabajados, retardos,
// faltas y horas por empleado en un rango de fechas. La exportación a
// Excel/PDF queda para el frontend (el JSON ya trae todo lo necesario para
// armar un CSV) o una siguiente iteración.
func (h *AsistenciaHandler) reporte(c *gin.Context) {
	hoy := time.Now().In(h.Location).Format("2006-01-02")
	inicio := c.DefaultQuery("inicio", hoy)
	fin := c.DefaultQuery("fin", hoy)

	fechaInicio, err1 := time.ParseInLocation("2006-01-02", inicio, h.Location)
	fechaFin, err2 := time.ParseInLocation("2006-01-02", fin, h.Location)
	if err1 != nil || err2 != nil || fechaFin.Before(fechaInicio) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rango de fechas inválido"})
		return
	}

	empleados, err := h.empleadosParaReporte(negocioID(c), c.Query("empleado_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar empleados"})
		return
	}

	hoyFecha := time.Now().In(h.Location)
	reportes := make([]gin.H, 0, len(empleados))

	for _, empleado := range empleados {
		registrosPorDia, err := h.registrosEnRango(empleado.id, inicio, fin)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar registros de asistencia"})
			return
		}

		detalle := []gin.H{}
		diasTrabajados, retardos, faltas := 0, 0, 0
		horasTrabajadas, horasExtra := 0.0, 0.0

		for dia := fechaInicio; !dia.After(fechaFin); dia = dia.AddDate(0, 0, 1) {
			fechaStr := dia.Format("2006-01-02")
			registrosDia := registrosPorDia[fechaStr]

			var entrada, salida *registroDelDia
			for i := range registrosDia {
				r := &registrosDia[i]
				if r.tipo == "entrada" && entrada == nil {
					entrada = r
				}
				if r.tipo == "salida" {
					salida = r
				}
			}

			aplicaTurno := empleado.turno != nil && contieneDia(empleado.turno.diasAplicables, int(dia.Weekday()))

			switch {
			case entrada != nil:
				diasTrabajados++
				estadoDia := entrada.estado
				if estadoDia == "retardo" {
					retardos++
				}
				var horasDia float64
				if salida != nil {
					horasDia = salida.horaReal.Sub(entrada.horaReal).Hours()
					horasTrabajadas += horasDia
					if empleado.turno != nil {
						duracion := duracionTurnoHoras(empleado.turno)
						if horasDia > duracion {
							horasExtra += horasDia - duracion
						}
					}
				}
				detalle = append(detalle, gin.H{
					"fecha": fechaStr, "estado": estadoDia, "horas_trabajadas": round2(horasDia),
				})
			case aplicaTurno && dia.Before(hoyFecha):
				faltas++
				detalle = append(detalle, gin.H{"fecha": fechaStr, "estado": "falta", "horas_trabajadas": 0})
			}
		}

		porcentajePuntualidad := 0.0
		if diasTrabajados > 0 {
			porcentajePuntualidad = round2(float64(diasTrabajados-retardos) / float64(diasTrabajados) * 100)
		}

		reportes = append(reportes, gin.H{
			"empleado_id": empleado.id,
			"nombre":      empleado.nombre,
			"resumen": gin.H{
				"dias_trabajados":        diasTrabajados,
				"retardos":               retardos,
				"faltas":                 faltas,
				"horas_trabajadas":       round2(horasTrabajadas),
				"horas_extra":            round2(horasExtra),
				"porcentaje_puntualidad": porcentajePuntualidad,
			},
			"detalle": detalle,
		})
	}

	c.JSON(http.StatusOK, gin.H{"inicio": inicio, "fin": fin, "empleados": reportes})
}

func (h *AsistenciaHandler) empleadosParaReporte(negID int, empleadoIDParam string) ([]reporteEmpleado, error) {
	query := `
		SELECT e.id, e.nombre_completo, t.hora_entrada, t.hora_salida, t.dias_aplicables
		FROM empleados e
		LEFT JOIN turnos t ON e.turno_id = t.id
		WHERE e.negocio_id = $1`
	args := []interface{}{negID}
	if empleadoIDParam != "" {
		empleadoID, err := strconv.Atoi(empleadoIDParam)
		if err != nil {
			return nil, err
		}
		query += " AND e.id = $2"
		args = append(args, empleadoID)
	}
	query += " ORDER BY e.nombre_completo ASC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	empleados := []reporteEmpleado{}
	for rows.Next() {
		var e reporteEmpleado
		var horaEntrada, horaSalida sql.NullTime
		var diasAplicables pq.Int64Array

		if err := rows.Scan(&e.id, &e.nombre, &horaEntrada, &horaSalida, &diasAplicables); err != nil {
			continue
		}
		if horaEntrada.Valid {
			e.turno = &turnoInfo{
				horaEntrada:    horaEntrada.Time.Format("15:04:05"),
				horaSalida:     horaSalida.Time.Format("15:04:05"),
				diasAplicables: toIntSlice(diasAplicables),
			}
		}
		empleados = append(empleados, e)
	}
	return empleados, nil
}

func (h *AsistenciaHandler) registrosEnRango(empleadoID int, inicio, fin string) (map[string][]registroDelDia, error) {
	rows, err := h.DB.Query(`
		SELECT fecha::text, tipo, hora_real, COALESCE(estado, '')
		FROM registros_asistencia
		WHERE empleado_id = $1 AND fecha BETWEEN $2 AND $3
		ORDER BY fecha ASC, hora_real ASC`, empleadoID, inicio, fin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	porDia := map[string][]registroDelDia{}
	for rows.Next() {
		var fecha string
		var r registroDelDia
		if err := rows.Scan(&fecha, &r.tipo, &r.horaReal, &r.estado); err != nil {
			continue
		}
		porDia[fecha] = append(porDia[fecha], r)
	}
	return porDia, nil
}

func contieneDia(dias []int, dia int) bool {
	for _, d := range dias {
		if d == dia {
			return true
		}
	}
	return false
}

func duracionTurnoHoras(turno *turnoInfo) float64 {
	entrada := atClock(time.Now(), turno.horaEntrada)
	salida := atClock(time.Now(), turno.horaSalida)
	return salida.Sub(entrada).Hours()
}

func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
