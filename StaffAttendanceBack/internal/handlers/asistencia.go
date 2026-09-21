package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"staffattendance/internal/rekognition"
)

// AsistenciaHandler cubrirá el marcaje de entrada/salida (SPEC.md sección
// 6.2) y los reportes (sección 6.4). Las rutas quedan registradas para fijar
// el contrato de la API; la lógica de puntualidad/retardo y de reportes se
// construye en la siguiente iteración, una vez resueltos los pendientes de
// la sección 8 (comportamiento sin internet, multi-negocio, nómina).
type AsistenciaHandler struct {
	DB  *sql.DB
	Rek *rekognition.Client
}

func NewAsistenciaHandler(db *sql.DB, rek *rekognition.Client) *AsistenciaHandler {
	return &AsistenciaHandler{DB: db, Rek: rek}
}

func (h *AsistenciaHandler) Register(router gin.IRouter) {
	router.POST("/asistencia/marcar", h.pendiente)
	router.GET("/asistencia/hoy", h.pendiente)
	router.GET("/reportes/asistencia", h.pendiente)
}

func (h *AsistenciaHandler) pendiente(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Pendiente de implementación — ver SPEC.md secciones 6.2/6.4 y 8",
	})
}
