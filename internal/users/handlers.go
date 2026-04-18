// internal/users/handlers.go
package users

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sime/shoply/internal/auth"
	"github.com/sime/shoply/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
    DB        *sql.DB
    JWTSecret string
}

func (h *Handler) Register(c *gin.Context) {
    var req struct {
        Name     string        `json:"name" binding:"required"`
        Email    string        `json:"email" binding:"required"`
        Password string        `json:"password" binding:"required"`
        Role     models.Role   `json:"role" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

    _, err := h.DB.Exec(
        "INSERT INTO users (name,email,password_hash,role,created_at,updated_at) VALUES ($1,$2,$3,$4,NOW(),NOW())",
        req.Name, req.Email, string(hashed), req.Role,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

func (h *Handler) Login(c *gin.Context) {
    var req struct {
        Email    string `json:"email" binding:"required"`
        Password string `json:"password" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user models.User
    row := h.DB.QueryRow("SELECT id, password_hash, role FROM users WHERE email=$1", req.Email)
    if err := row.Scan(&user.ID, &user.PasswordHash, &user.Role); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }

    token, _ := auth.GenerateJWT(user.ID.String(), string(user.Role), h.JWTSecret)
    c.JSON(http.StatusOK, gin.H{"token": token})
}