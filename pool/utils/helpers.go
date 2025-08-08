package utils

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/google/uuid"
)

func GenerateUUID() string {
	return uuid.New().String()
}

func GenerateRandomHex(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

func ValidateUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
