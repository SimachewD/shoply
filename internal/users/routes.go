package users

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sime/shoply/internal/auth"
	"github.com/sime/shoply/internal/utils"
)

func UserRoutes(r *gin.Engine, db *sql.DB, redisClient *redis.Client) {
	repo := NewRepository(db)
	service := NewService(repo, utils.GetEnv("JWT_SECRET", ""))
	handler := NewHandler(service)

	// auth routes
	authRoutes := r.Group("/api/auth")
	{
		authRoutes.POST("/register", handler.Register)
		authRoutes.POST("/login", handler.Login)
		authRoutes.POST("/logout", handler.Logout)
	}

	// user routes
	userRoutes := r.Group("/api/user")
	userRoutes.Use(auth.AuthMiddleware())
	{
		userRoutes.GET("/profile/:id", handler.GetProfile)
		userRoutes.PATCH("/profile/:id", handler.UpdateProfile)
	}

	// admin routes
	admin := r.Group("/api/admin")
	admin.Use(auth.AuthMiddleware())
	admin.Use(auth.RequireRole("admin"))
	{
		admin.GET("/users", handler.GetUsers)
		admin.DELETE("/users/:id", handler.DeleteUser)
		admin.PATCH("/users/:id/role", handler.ChangeUserRole)
		admin.GET("/users/:id", handler.GetProfile)
		admin.PATCH("/users/:id", handler.UpdateProfile)
	}
}
