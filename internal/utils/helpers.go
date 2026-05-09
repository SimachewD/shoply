package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

func GetEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func HashToken(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}