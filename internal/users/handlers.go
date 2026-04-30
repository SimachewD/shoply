package users

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sime/shoply/internal/auth"
	"github.com/sime/shoply/internal/models"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := h.service.Register(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
        "user": user,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	access, refresh, user, err := h.service.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 🍪 set refresh token cookie
	c.SetCookie(
		"refresh_token",
		refresh,
		int(auth.RefreshTokenTTL.Seconds()),
		"/",
		"",
		true, // Secure
		true, // HttpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": access,
		"user":         user,
	})
}

//
// 🔁 REFRESH
//

func (h *Handler) Refresh(c *gin.Context) {
	token, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}

	access, newRefresh, err := h.service.Refresh(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh"})
		return
	}

	// rotate cookie
	c.SetCookie(
		"refresh_token",
		newRefresh,
		int(auth.RefreshTokenTTL.Seconds()),
		"/",
		"",
		true,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": access,
	})
}

//
// 🚪 LOGOUT
//

func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Cookie("refresh_token")

	err := h.service.Logout(token)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
        })
        return
    }

	// clear cookie
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out",
	})
}

// admin routes
func (h *Handler) GetUsers(c *gin.Context) {
    users, err := h.service.GetAllUsers()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "users": users,
    })
}

func (h *Handler) GetPendingRequests(c *gin.Context) {
	reqs, err := h.service.GetPendingRequests()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, reqs)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	user, err := h.service.DeleteUser(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted", "user": user})
}

func (h *Handler) ChangeUserRole(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Role models.Role `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.ChangeUserRole(id, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// user routes
func (h *Handler) GetProfile(c *gin.Context) {
    id := c.GetString("userID")
    
	user, err := h.service.GetUserByID(id)
	if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
		})
		return
	}
    
	c.JSON(http.StatusOK, user)
}


func (h *Handler) GetUserByEmail(c *gin.Context) {
    email := c.Param("email")

	user, err := h.service.GetUserByEmail(email)
	if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
		})
		return
	}
    
	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
    id := c.GetString("userID")

    var req struct {
        Name  string `json:"name" binding:"required"`
        Email string `json:"email" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    user, err := h.service.GetUserByID(id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
        })
        return
    }

    user.Name = req.Name
    user.Email = req.Email

    user, err = h.service.UpdateUser(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "profile updated", "user": user})
}

func (h *Handler) RequestSeller(c *gin.Context) {
	userID := c.GetString("userID")

	err := h.service.CreateSellerRequest(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "seller request submitted",
	})
}