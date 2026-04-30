package utils

import (
	"crypto/sha256"
	"encoding/hex"
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

func HashToken(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}