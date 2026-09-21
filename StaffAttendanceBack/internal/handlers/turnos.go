package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"staffattendance/internal/models"
)

type TurnosHandler struct {
	DB *sql.DB
}

func NewTurnosHandler(db *sql.DB) *TurnosHandler {
	return &TurnosHandler{DB: db}
}

func (h *TurnosHandler) Register(router gin.IRouter) {
	router.POST("/turnos", h.crear)
	router.GET("/turnos", h.listar)
}

func (h *TurnosHandler) crear(c *gin.Context) {
	var input struct {
		Nombre                   string `json:"nombre" binding:"required"`
		HoraEntrada              string `json:"hora_entrada" binding:"required"`
		HoraSalida               string `json:"hora_salida" binding:"required"`
		ToleranciaRetardoMinutos int    `json:"tolerancia_retardo_minutos"`
		DiasAplicables           []int  `json:"dias_aplicables"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	if input.ToleranciaRetardoMinutos == 0 {
		input.ToleranciaRetardoMinutos = 10
	}
	if len(input.DiasAplicables) == 0 {
		input.DiasAplicables = []int{1, 2, 3, 4, 5}
	}

	var id int
	err := h.DB.QueryRow(
		`INSERT INTO turnos (negocio_id, nombre, hora_entrada, hora_salida, tolerancia_retardo_minutos, dias_aplicables)
         VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		negocioID(c), input.Nombre, input.HoraEntrada, input.HoraSalida, input.ToleranciaRetardoMinutos, pq.Array(input.DiasAplicables),
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el turno"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *TurnosHandler) listar(c *gin.Context) {
	rows, err := h.DB.Query(
		`SELECT id, nombre, hora_entrada, hora_salida, tolerancia_retardo_minutos, dias_aplicables
         FROM turnos WHERE negocio_id = $1 ORDER BY nombre ASC`,
		negocioID(c),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar turnos"})
		return
	}
	defer rows.Close()

	turnos := []models.Turno{}
	for rows.Next() {
		var t models.Turno
		var horaEntrada, horaSalida time.Time
		var dias pq.Int64Array
		if err := rows.Scan(&t.ID, &t.Nombre, &horaEntrada, &horaSalida, &t.ToleranciaRetardoMinutos, &dias); err != nil {
			continue
		}
		t.HoraEntrada = horaEntrada.Format("15:04:05")
		t.HoraSalida = horaSalida.Format("15:04:05")
		t.DiasAplicables = toIntSlice(dias)
		turnos = append(turnos, t)
	}

	c.JSON(http.StatusOK, turnos)
}
