package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"staffattendance/internal/middleware"
)

type AuthHandler struct {
	DB        *sql.DB
	JWTSecret []byte
}

func NewAuthHandler(db *sql.DB, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{DB: db, JWTSecret: jwtSecret}
}

// RegisterPublic monta /login, sin autenticación.
func (h *AuthHandler) RegisterPublic(router gin.IRouter) {
	router.POST("/login", h.login)
}

// RegisterProtected monta /usuarios/registro: solo un admin ya autenticado
// puede dar de alta más usuarios, y siempre dentro de su propio negocio (el
// negocio_id sale del token, nunca del body, para no permitir que un
// negocio cree usuarios en otro).
func (h *AuthHandler) RegisterProtected(router gin.IRouter) {
	router.POST("/usuarios/registro", h.crearUsuario)
}

func (h *AuthHandler) crearUsuario(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Rol      string `json:"rol"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	if input.Rol == "" {
		input.Rol = "admin"
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}

	_, err = h.DB.Exec(
		`INSERT INTO usuarios (negocio_id, username, password_hash, rol) VALUES ($1, $2, $3, $4)`,
		negocioID(c), input.Username, string(hashedPassword), input.Rol,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el usuario"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Usuario creado exitosamente"})
}

func (h *AuthHandler) login(c *gin.Context) {
	var creds struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	var id, negID int
	var passwordHash, rol, negocioNombre, negocioSlug string
	err := h.DB.QueryRow(
		`SELECT u.id, u.password_hash, u.rol, u.negocio_id, n.nombre, n.slug
         FROM usuarios u
         JOIN negocios n ON n.id = u.negocio_id
         WHERE u.username = $1`,
		creds.Username,
	).Scan(&id, &passwordHash, &rol, &negID, &negocioNombre, &negocioSlug)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no existe"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error de base de datos"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(creds.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Contraseña incorrecta"})
		return
	}

	tokenString, err := generarToken(h.JWTSecret, id, negID, rol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":          tokenString,
		"rol":            rol,
		"username":       creds.Username,
		"negocio_id":     negID,
		"negocio_nombre": negocioNombre,
		"negocio_slug":   negocioSlug,
	})
}

func generarToken(jwtSecret []byte, userID, negocioID int, rol string) (string, error) {
	claims := &middleware.Claims{
		UserID:    userID,
		NegocioID: negocioID,
		Rol:       rol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
