// internal/users/routes.go
package users

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB, redisClient *redis.Client) {
    h := &Handler{DB: db, JWTSecret: "REPLACE_WITH_ENV_JWT_SECRET"}

    r.POST("/api/register", h.Register)
    r.POST("/api/login", h.Login)
}