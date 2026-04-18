// internal/config/config.go
package config

import (
	"fmt"

	"github.com/sime/shoply/internal/utils"
)

type Config struct {
	PostgresURL string
	RedisUrl    string
}

func LoadConfig() *Config {
	pgHost := utils.GetEnv("DB_HOST", "localhost")
	pgPort := utils.GetEnv("DB_PORT", "54432")
	pgUser := utils.GetEnv("DB_USER", "shoply_user")
	pgPass := utils.GetEnv("DB_PASSWORD", "3165")
	pgDB := utils.GetEnv("DB_NAME", "shoply_db")

	redisUrl := utils.GetEnv("REDIS_URL", "redis://localhost:6379/0")

	return &Config{
		PostgresURL: fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
			pgUser, pgPass, pgHost, pgPort, pgDB,
		),
		RedisUrl: redisUrl,
	}
}