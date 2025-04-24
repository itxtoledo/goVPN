package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Message represents a message in the system
type Message struct {
	Type          string `json:"type"`
	RoomID        string `json:"room_id,omitempty"`
	RoomName      string `json:"room_name,omitempty"`
	PublicKey     string `json:"public_key,omitempty"`
	Password      string `json:"password,omitempty"`
	DestinationID string `json:"destination_id,omitempty"`
	TargetID      string `json:"target_id,omitempty"`
	Username      string `json:"username,omitempty"`
	Offer         string `json:"offer,omitempty"`
	Answer        string `json:"answer,omitempty"`
	Candidate     string `json:"candidate,omitempty"`
	Data          []byte `json:"data,omitempty"`
	Signature     string `json:"signature,omitempty"`
	MessageID     string `json:"message_id,omitempty"` // ID único para rastreamento de mensagens
}

// Room represents a network or room
type Room struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Password    string `json:"password,omitempty"`
	ClientCount int    `json:"client_count"`
}

// GenerateMessageID gera um ID aleatório em formato hexadecimal com base no comprimento especificado
func GenerateMessageID(length int) (string, error) {
	// Determine quantos bytes precisamos para gerar o ID
	byteLength := (length + 1) / 2 // arredondamento para cima para garantir bytes suficientes

	bytes := make([]byte, byteLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("falha ao gerar bytes aleatórios: %w", err)
	}

	// Converte para hexadecimal e limita ao comprimento desejado
	id := hex.EncodeToString(bytes)
	if len(id) > length {
		id = id[:length]
	}

	return id, nil
}
