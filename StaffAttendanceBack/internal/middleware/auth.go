package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    int    `json:"user_id"`
	NegocioID int    `json:"negocio_id"`
	Rol       string `json:"rol"`
	jwt.RegisteredClaims
}

// Auth reutiliza el esquema JWT de GuarderiaBiometric: valida el header
// Authorization y expone user_id/negocio_id/rol en el contexto de Gin. El
// negocio_id es lo que aísla los datos de cada negocio (multi-negocio,
// SPEC.md sección 8).
func Auth(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token requerido"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("negocio_id", claims.NegocioID)
		c.Set("rol", claims.Rol)
		c.Next()
	}
}

// RequireRol restringe una ruta a ciertos roles. Va después de Auth() en la
// cadena, así que ya puede leer "rol" del contexto. Dos perfiles hoy:
// "admin" (todo) y "kiosco" (una tablet del mostrador que solo puede marcar
// asistencia — no gestiona empleados, turnos, reportes ni usuarios).
func RequireRol(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolValue, _ := c.Get("rol")
		rol, _ := rolValue.(string)

		for _, permitido := range roles {
			if rol == permitido {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Tu perfil no tiene permiso para esta acción"})
	}
}
