package handlers

import (
	"database/sql"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"staffattendance/internal/models"
	"staffattendance/internal/rekognition"
)

type EmpleadosHandler struct {
	DB  *sql.DB
	Rek *rekognition.Client
}

func NewEmpleadosHandler(db *sql.DB, rek *rekognition.Client) *EmpleadosHandler {
	return &EmpleadosHandler{DB: db, Rek: rek}
}

func (h *EmpleadosHandler) Register(router gin.IRouter) {
	router.POST("/empleados", h.crear)
	router.GET("/empleados", h.listar)
	router.PATCH("/empleados/:id/estado", h.cambiarEstado)
	router.POST("/empleados/:id/enrolar", h.enrolar)
}

func (h *EmpleadosHandler) crear(c *gin.Context) {
	var input struct {
		NombreCompleto string `json:"nombre_completo" binding:"required"`
		Puesto         string `json:"puesto"`
		Departamento   string `json:"departamento"`
		TurnoID        *int   `json:"turno_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	var id int
	err := h.DB.QueryRow(
		`INSERT INTO empleados (nombre_completo, puesto, departamento, turno_id)
         VALUES ($1, $2, $3, $4) RETURNING id`,
		input.NombreCompleto, input.Puesto, input.Departamento, input.TurnoID,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el empleado"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *EmpleadosHandler) listar(c *gin.Context) {
	rows, err := h.DB.Query(
		`SELECT id, nombre_completo, puesto, departamento, turno_id, estado,
                COALESCE(rekognition_face_id, ''), fecha_alta
         FROM empleados ORDER BY nombre_completo ASC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar empleados"})
		return
	}
	defer rows.Close()

	empleados := []models.Empleado{}
	for rows.Next() {
		var e models.Empleado
		if err := rows.Scan(&e.ID, &e.NombreCompleto, &e.Puesto, &e.Departamento, &e.TurnoID, &e.Estado, &e.RekognitionFaceID, &e.FechaAlta); err != nil {
			continue
		}
		empleados = append(empleados, e)
	}

	c.JSON(http.StatusOK, empleados)
}

func (h *EmpleadosHandler) cambiarEstado(c *gin.Context) {
	empleadoID := c.Param("id")

	var input struct {
		Estado string `json:"estado" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || (input.Estado != "activo" && input.Estado != "inactivo") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Estado inválido, use 'activo' o 'inactivo'"})
		return
	}

	result, err := h.DB.Exec(`UPDATE empleados SET estado = $1 WHERE id = $2`, input.Estado, empleadoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el estado"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Empleado no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"estado": input.Estado})
}

// enrolar implementa la sección 6.1: captura el rostro del empleado, lo
// indexa en la colección de Rekognition de empleados y guarda el
// rekognition_face_id devuelto (reutiliza el flujo de IndexFaces de
// GuarderiaBiometric, ver SPEC.md sección 3).
func (h *EmpleadosHandler) enrolar(c *gin.Context) {
	empleadoID := c.Param("id")

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

	var nombreCompleto string
	if err := h.DB.QueryRow(`SELECT nombre_completo FROM empleados WHERE id = $1`, empleadoID).Scan(&nombreCompleto); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Empleado no encontrado"})
		return
	}

	externalImageID := strings.ReplaceAll(nombreCompleto, " ", "_") + "_" + empleadoID
	faceID, err := h.Rek.IndexFace(c.Request.Context(), imageBytes, externalImageID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	_, err = h.DB.Exec(`UPDATE empleados SET rekognition_face_id = $1 WHERE id = $2`, faceID, empleadoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar el rostro del empleado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rekognition_face_id": faceID})
}
