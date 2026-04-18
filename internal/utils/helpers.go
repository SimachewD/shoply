package utils

import (
	"fmt"
	"os"
)

func GetEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		fmt.Println("Using environment variable for", key, ":", val)
		return val
	}
	return defaultVal
}
