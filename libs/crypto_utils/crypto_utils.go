package crypto_utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"
)

// ParsePublicKey parses a base64 encoded public key
func ParsePublicKey(keyStr string) (ed25519.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		log.Printf("Error decoding public key from base64: %v", err)
		return nil, err
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		log.Printf("Invalid public key size: got %d bytes, expected %d bytes", len(keyBytes), ed25519.PublicKeySize)
		return nil, errors.New("invalid public key size")
	}

	return ed25519.PublicKey(keyBytes), nil
}

// Encrypt encrypts data using AES-GCM
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	result := append(nonce, ciphertext...)

	return result, nil
}

// Decrypt decrypts data using AES-GCM
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(ciphertext) < 12 {
		return nil, errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	nonce := ciphertext[:12]
	ciphertext = ciphertext[12:]

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
