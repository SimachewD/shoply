package users

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sime/shoply/internal/auth"
	"github.com/sime/shoply/internal/models"
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
		authRoutes.POST("/refresh", handler.Refresh)
		authRoutes.POST("/logout", handler.Logout)
		authRoutes.GET("/me", auth.AuthMiddleware(), handler.GetProfile)
		authRoutes.PATCH("/me", auth.AuthMiddleware(), handler.UpdateProfile)
	}

	// admin routes
	adminRoutes := r.Group("/api/admin")

	adminRoutes.Use(auth.AuthMiddleware())
	adminRoutes.Use(auth.RequireRole(string(models.RoleAdmin)))
	{
		adminRoutes.GET("/users", handler.GetUsers)
		adminRoutes.GET("/deleted-users", handler.GetDeletedUsers)
		adminRoutes.PATCH("/users/:id/role", handler.ChangeRole)
		adminRoutes.PATCH("/users/:id/suspend", handler.SuspendUser)
		adminRoutes.PATCH("/users/:id/ban", handler.BanUser)
		adminRoutes.PATCH("/users/:id/activate", handler.ActivateUser)
		adminRoutes.DELETE("/users/:id/delete", handler.DeleteUser)
		adminRoutes.PATCH("/users/:id/restore", handler.RestoreUser)
		adminRoutes.GET("/users/:id/audit-logs", handler.GetUserAuditLogs)
	}

	// user routes
	userRoutes := r.Group("/api/user")
	userRoutes.Use(auth.AuthMiddleware())
	{
		// TODO: implement authenticated user routes
	}

	// public routes
	public := r.Group("/api")
	{
		public.GET("/users/:id", handler.GetProfile)
	}
}
