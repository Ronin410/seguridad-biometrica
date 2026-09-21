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
