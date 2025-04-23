package models

import (
	"encoding/json"
)

// Room representa uma sala virtual na rede VPN
type Room struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Password    string `json:"password,omitempty"` // Omitido em respostas públicas
	ClientCount int    `json:"clientCount"`
	Status      string `json:"status,omitempty"`    // Campo apenas para o cliente
	IsCreator   bool   `json:"isCreator,omitempty"` // Campo apenas para o cliente
}

// Message representa uma mensagem websocket
type Message struct {
	Type          string          `json:"type"`
	RoomID        string          `json:"roomID,omitempty"`
	RoomName      string          `json:"roomName,omitempty"`
	Password      string          `json:"password,omitempty"`
	PublicKey     string          `json:"publicKey,omitempty"`
	TargetID      string          `json:"targetID,omitempty"`
	Signature     string          `json:"signature,omitempty"`
	Data          json.RawMessage `json:"data,omitempty"`
	Candidate     json.RawMessage `json:"candidate,omitempty"`
	Offer         json.RawMessage `json:"offer,omitempty"`
	Answer        json.RawMessage `json:"answer,omitempty"`
	DestinationID string          `json:"destinationID,omitempty"`
}

// NetworkPacket representa um pacote de rede tunelado pela VPN
type NetworkPacket struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Protocol    string `json:"protocol"` // tcp, udp, icmp
	Port        int    `json:"port,omitempty"`
	Data        []byte `json:"data"`
	Encrypted   bool   `json:"encrypted"`
}

// ClientInfo representa informações sobre um cliente conectado
type ClientInfo struct {
	ID        string `json:"id"`
	Connected bool   `json:"connected"`
	P2PStatus string `json:"p2pStatus"` // "direct", "relayed", "failed"
	Latency   int64  `json:"latency"`   // em milissegundos
}

// RoomListResponse é utilizado na API para listar salas disponíveis
type RoomListResponse struct {
	Rooms []Room `json:"rooms"`
}

// ErrorResponse representa uma resposta de erro
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
	Code    int    `json:"code"`
}
