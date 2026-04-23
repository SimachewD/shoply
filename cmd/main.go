package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sime/shoply/config"
	"github.com/sime/shoply/internal/database"
	"github.com/sime/shoply/internal/users"
    "github.com/gin-contrib/cors"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from system environment")
	}

	cfg := config.LoadConfig()

	fmt.Println("Postgres URL:", cfg.PostgresURL)
	fmt.Println("Redis URL:", cfg.RedisUrl)

	// 🟢 1. Connect DB
	db, err := database.ConnectPostgres(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("failed to connect to Postgres: %v", err)
	}

	// 🟢 2. Connect Redis
	rdb, err := database.ConnectRedis(cfg.RedisUrl)
	if err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	// 🟢 3. CREATE REPOSITORY
	userRepo := users.NewRepository(db)

	// 🟢 4. SEED ADMIN (IMPORTANT PLACE)
	users.SeedAdmin(userRepo)

	// 🟢 5. START SERVER
	r := gin.Default()

    // 🟢 6. Enable CORS
    r.Use(cors.Default())

	users.UserRoutes(r, db, rdb)

	log.Printf("Shoply running on port %s", ":8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}