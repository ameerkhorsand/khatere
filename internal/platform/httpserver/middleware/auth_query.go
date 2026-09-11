package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthRequiredQuery checks a JWT passed as ?token=... on the URL.
// Use this only for routes that a browser cannot attach a header to,
// such as a WebSocket handshake. Every other route should keep using
// AuthRequired.
func AuthRequiredQuery(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken := c.Query("token")
		if rawToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(rawToken, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		accountID := claims.Subject
		if accountID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token missing subject"})
			return
		}

		c.Set("account_id", accountID)
		c.Set("account_type", claims.AccountType)
		c.Next()
	}
}
