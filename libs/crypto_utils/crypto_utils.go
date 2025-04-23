package crypto_utils

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/itxtoledo/govpn/libs/models"
)

// GenerateRSAKeys generates an RSA key pair and returns both keys as base64 encoded strings
func GenerateRSAKeys() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Marshal the public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	publicKeyStr := base64.StdEncoding.EncodeToString(pubKeyBytes)

	// Marshal the private key
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyStr := base64.StdEncoding.EncodeToString(privKeyBytes)

	return privateKeyStr, publicKeyStr, nil
}

// SignMessage signs a message with the private key
func SignMessage(msg models.Message, privateKey *rsa.PrivateKey) (string, error) {
	msgCopy := msg
	msgCopy.Signature = ""
	data, err := json.Marshal(msgCopy)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// VerifySignature verifies a message signature using the public key
func VerifySignature(msg models.Message, pubKey *rsa.PublicKey) bool {
	if msg.Signature == "" {
		return false
	}

	sigBytes, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		return false
	}

	msgCopy := msg
	msgCopy.Signature = ""
	data, err := json.Marshal(msgCopy)
	if err != nil {
		return false
	}
	hash := sha256.Sum256(data)

	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes)
	return err == nil
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

// GenerateID creates a unique ID
func GenerateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateRoomID creates a SHA-256 hash based on room name, salt, and timestamp
func GenerateRoomID(roomName string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	input := fmt.Sprintf("%s:%s:%d", roomName, base64.StdEncoding.EncodeToString(salt), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
