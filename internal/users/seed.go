package users

import (
	"log"

	"github.com/sime/shoply/internal/models"
	"github.com/sime/shoply/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

func SeedAdmin(repo *Repository) {
	email := utils.GetEnv("ADMIN_EMAIL", "admin@shoply.com")
	password := utils.GetEnv("ADMIN_PASSWORD", "admin123")

	// check if admin exists
	_, err := repo.GetUserByEmail(email)
	if err == nil {
		log.Println("Admin already exists")
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	admin := &models.User{
		Name:         "Admin",
		Email:        email,
		PasswordHash: string(hashed),
		Role:         models.RoleAdmin,
	}

	_, err = repo.Register(admin)
	if err != nil {
		log.Println("Failed to seed admin:", err)
		return
	}

	log.Println("Admin created: admin@shoply.com / admin123")
}