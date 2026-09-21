package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"staffattendance/internal/rekognition"
)

type NegociosHandler struct {
	DB        *sql.DB
	Rek       rekognition.FaceRecognizer
	JWTSecret []byte
}

func NewNegociosHandler(db *sql.DB, rek rekognition.FaceRecognizer, jwtSecret []byte) *NegociosHandler {
	return &NegociosHandler{DB: db, Rek: rek, JWTSecret: jwtSecret}
}

// Register monta /negocios, sin autenticación: es el alta de un nuevo
// negocio (multi-negocio, SPEC.md sección 8) junto con su primer usuario
// admin. Cualquier otro usuario se crea luego vía /usuarios/registro, ya
// autenticado dentro de ese negocio.
func (h *NegociosHandler) Register(router gin.IRouter) {
	router.POST("/negocios", h.crear)
}

var slugInvalido = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(nombre string) string {
	slug := slugInvalido.ReplaceAllString(strings.ToLower(nombre), "-")
	return strings.Trim(slug, "-")
}

func (h *NegociosHandler) crear(c *gin.Context) {
	var input struct {
		NegocioNombre string `json:"negocio_nombre" binding:"required"`
		NegocioSlug   string `json:"negocio_slug"`
		Username      string `json:"username" binding:"required"`
		Password      string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	slug := input.NegocioSlug
	if slug == "" {
		slug = slugify(input.NegocioNombre)
	}
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se pudo derivar un identificador (slug) del nombre del negocio"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar la transacción"})
		return
	}
	defer tx.Rollback()

	var negID int
	err = tx.QueryRow(`INSERT INTO negocios (nombre, slug) VALUES ($1, $2) RETURNING id`, input.NegocioNombre, slug).Scan(&negID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Ese identificador (slug) ya está en uso, elige otro"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el negocio"})
		return
	}

	var userID int
	err = tx.QueryRow(
		`INSERT INTO usuarios (negocio_id, username, password_hash, rol) VALUES ($1, $2, $3, 'admin') RETURNING id`,
		negID, input.Username, string(hashedPassword),
	).Scan(&userID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Ese nombre de usuario ya está en uso"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el usuario administrador"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo confirmar el alta del negocio"})
		return
	}

	// No es fatal: si AWS no está listo todavía, la colección se puede crear
	// más tarde (se reintenta sola al enrolar el primer empleado).
	collectionID := rekognition.CollectionID(negID)
	if err := h.Rek.EnsureCollection(c.Request.Context(), collectionID); err != nil {
		log.Printf("Aviso: no se pudo preparar la colección de Rekognition del negocio %d (%v)", negID, err)
	}

	tokenString, err := generarToken(h.JWTSecret, userID, negID, "admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Negocio creado, pero no se pudo generar el token de acceso"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"negocio_id":     negID,
		"negocio_nombre": input.NegocioNombre,
		"negocio_slug":   slug,
		"token":          tokenString,
	})
}
