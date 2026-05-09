package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sime/shoply/internal/response"
	"github.com/sime/shoply/internal/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			response.Abort(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Abort(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization format")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := ValidateAccessToken(
			tokenString,
			utils.GetEnv("JWT_SECRET", "your_jwt_secret"),
		)
		if err != nil {
			response.Abort(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired access token")
			return
		}

		// ✅ set context
		c.Set("userID", claims.UserID.String())
		c.Set("role", claims.Role)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString("role")

		for _, role := range roles {
			if role == userRole {
				c.Next()
				return
			}
		}

		response.Abort(c, http.StatusForbidden, "FORBIDDEN", "forbidden")
	}
}