// cmd/shoply/main.go
package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sime/shoply/config"
	"github.com/sime/shoply/internal/database"
	"github.com/sime/shoply/internal/users"
)

func main() {
    // Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from system environment")
	}
	
	// ctx := context.Background()
    
    // Load configuration
    cfg := config.LoadConfig()

    fmt.Println("Postgres URL:", cfg.PostgresURL)
    fmt.Println("Redis URL:", cfg.RedisUrl)

    // Connect to Postgres
    db, err := database.ConnectPostgres(cfg.PostgresURL)
    if err != nil {
        log.Fatalf("failed to connect to Postgres: %v", err)
    }

    // Connect to Redis
    rdb, err := database.ConnectRedis(cfg.RedisUrl)
    if err != nil {
        log.Fatalf("failed to connect to Redis: %v", err)
    }

    // Initialize router
    r := gin.Default()

    // User routes
    users.RegisterRoutes(r, db, rdb)

    // Start server
    log.Printf("Shoply running on port %s", ":8000")
    if err := r.Run(":8000"); err != nil {
        log.Fatalf("server failed: %v", err)
    }
}