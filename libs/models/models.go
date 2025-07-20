package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

// PIN validation constants
const (
	// DefaultPINPattern is the default PIN validation pattern: exactly 4 numeric digits
	DefaultPINPattern = `^\d{4}$`
)

// Network represents a network or network
type Network struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PIN    string `json:"pin"`
	ClientCount int    `json:"client_count"`
}

// Helper functions

// GenerateMessageID generates a random ID in hexadecimal format based on the specified length
func GenerateMessageID() (string, error) {
	return GenerateRandomID(8)
}

func GenerateNetworkID() string {
	id, err := GenerateRandomID(16)

	if err != nil {
		// Fall back to a timestamp-based ID if random generation fails
		return fmt.Sprintf("%06x", time.Now().UnixNano()%0xFFFFFF)
	}

	return id
}

// GenerateRandomID generates a random ID in hexadecimal format with the desired length
func GenerateRandomID(length int) (string, error) {
	// Determine how many bytes we need to generate the ID
	byteLength := (length + 1) / 2 // round up to ensure sufficient bytes

	bytes := make([]byte, byteLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hexadecimal and limit to desired length
	id := hex.EncodeToString(bytes)
	if len(id) > length {
		id = id[:length]
	}

	return id, nil
}

// PINRegex returns a compiled regex for the default PIN pattern
func PINRegex() (*regexp.Regexp, error) {
	return regexp.Compile(DefaultPINPattern)
}

// ValidatePIN checks if a PIN matches the default PIN pattern
func ValidatePIN(pin string) bool {
	regex, err := PINRegex()
	if err != nil {
		return false
	}
	return regex.MatchString(pin)
}