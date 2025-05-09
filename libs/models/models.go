package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

// MessageType defines the type of messages that can be sent between client and server
type MessageType string

// Message type constants
const (
	// Client to server message types
	TypeCreateRoom     MessageType = "CreateRoom"
	TypeJoinRoom       MessageType = "JoinRoom"
	TypeConnectRoom    MessageType = "ConnectRoom"    // New: Conectar a uma sala previamente associada
	TypeDisconnectRoom MessageType = "DisconnectRoom" // New: Desconectar de uma sala sem sair dela
	TypeLeaveRoom      MessageType = "LeaveRoom"
	TypeKick           MessageType = "Kick"
	TypeRename         MessageType = "Rename"
	TypePing           MessageType = "Ping"         // Added for connection testing
	TypeGetUserRooms   MessageType = "GetUserRooms" // New: Get all rooms a user has joined

	// Server to client message types
	TypeError            MessageType = "Error"
	TypeRoomCreated      MessageType = "RoomCreated"
	TypeRoomJoined       MessageType = "RoomJoined"
	TypeRoomConnected    MessageType = "RoomConnected"    // New: Confirmação de conexão à sala
	TypeRoomDisconnected MessageType = "RoomDisconnected" // New: Confirmação de desconexão da sala
	TypeRoomDeleted      MessageType = "RoomDeleted"
	TypeRoomRenamed      MessageType = "RoomRenamed"
	TypePeerJoined       MessageType = "PeerJoined"
	TypePeerLeft         MessageType = "PeerLeft"
	TypePeerConnected    MessageType = "PeerConnected"    // New: Notificação de peer conectado
	TypePeerDisconnected MessageType = "PeerDisconnected" // New: Notificação de peer desconectado
	TypeKicked           MessageType = "Kicked"
	TypeKickSuccess      MessageType = "KickSuccess"
	TypeRenameSuccess    MessageType = "RenameSuccess"
	TypeDeleteSuccess    MessageType = "DeleteSuccess"
	TypeServerShutdown   MessageType = "ServerShutdown"
	TypeUserRooms        MessageType = "UserRooms" // New: Response with all rooms a user has joined
)

// Password validation constants
const (
	// DefaultPasswordPattern é o padrão de validação de senha padrão: exatamente 4 dígitos numéricos
	DefaultPasswordPattern = `^\d{4}$`
)

// SignalingMessage represents the wrapper structure for WebSocket communication
type SignalingMessage struct {
	ID      string      `json:"message_id"`
	Type    MessageType `json:"type"`
	Payload []byte      `json:"payload"`
}

// BaseRequest contains common fields used in all messages from client to server
type BaseRequest struct {
	PublicKey string `json:"public_key"` // Base64-encoded Ed25519 public key
}

// ErrorResponse is sent when an error occurs
type ErrorResponse struct {
	Error string `json:"error"`
}

// Room represents a network or room
type Room struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Password    string `json:"password,omitempty"`
	ClientCount int    `json:"client_count"`
}

// Event-specific request structs

// CreateRoomRequest represents a request to create a new room
type CreateRoomRequest struct {
	BaseRequest
	RoomName string `json:"room_name"`
	Password string `json:"password"`
}

// CreateRoomResponse represents a response to a room creation request
type CreateRoomResponse struct {
	RoomID    string `json:"room_id"`
	RoomName  string `json:"room_name"`
	Password  string `json:"password"`
	PublicKey string `json:"public_key"`
}

// JoinRoomRequest represents a request to join an existing room
type JoinRoomRequest struct {
	BaseRequest
	RoomID   string `json:"room_id"`
	Password string `json:"password"`
	Username string `json:"username,omitempty"`
}

// JoinRoomResponse represents a response to a room join request
type JoinRoomResponse struct {
	RoomID   string `json:"room_id"`
	RoomName string `json:"room_name"`
}

// ConnectRoomRequest represents a request to connect to a previously joined room
type ConnectRoomRequest struct {
	BaseRequest
	RoomID   string `json:"room_id"`
	Username string `json:"username,omitempty"`
}

// ConnectRoomResponse represents a response to a room connection request
type ConnectRoomResponse struct {
	RoomID   string `json:"room_id"`
	RoomName string `json:"room_name"`
}

// DisconnectRoomRequest represents a request to disconnect from a room (but stay joined)
type DisconnectRoomRequest struct {
	BaseRequest
	RoomID string `json:"room_id"`
}

// DisconnectRoomResponse represents a response to a room disconnect request
type DisconnectRoomResponse struct {
	RoomID string `json:"room_id"`
}

// LeaveRoomRequest represents a request to leave a room
type LeaveRoomRequest struct {
	BaseRequest
	RoomID string `json:"room_id"`
}

// LeaveRoomResponse confirms a client has left a room
type LeaveRoomResponse struct {
	RoomID string `json:"room_id"`
}

// Room management structs

// KickRequest represents a request to kick a user from a room
type KickRequest struct {
	BaseRequest
	RoomID   string `json:"room_id"`
	TargetID string `json:"target_id"`
}

// KickResponse confirms a user has been kicked
type KickResponse struct {
	RoomID   string `json:"room_id"`
	TargetID string `json:"target_id"`
}

// RenameRequest represents a request to rename a room
type RenameRequest struct {
	BaseRequest
	RoomID   string `json:"room_id"`
	RoomName string `json:"room_name"`
}

// RenameResponse confirms a room has been renamed
type RenameResponse struct {
	RoomID   string `json:"room_id"`
	RoomName string `json:"room_name"`
}

// Peer notification structs

// PeerJoinedNotification notifies that a peer has joined the room
type PeerJoinedNotification struct {
	RoomID    string `json:"room_id"`
	PublicKey string `json:"public_key"`
	Username  string `json:"username,omitempty"`
}

// PeerLeftNotification notifies that a peer has left the room
type PeerLeftNotification struct {
	RoomID    string `json:"room_id"`
	PublicKey string `json:"public_key"`
}

// PeerConnectedNotification notifies that a peer has connected to the room
type PeerConnectedNotification struct {
	RoomID    string `json:"room_id"`
	PublicKey string `json:"public_key"`
	Username  string `json:"username,omitempty"`
}

// PeerDisconnectedNotification notifies that a peer has disconnected from the room (but not left)
type PeerDisconnectedNotification struct {
	RoomID    string `json:"room_id"`
	PublicKey string `json:"public_key"`
}

// RoomDeletedNotification notifies that a room has been deleted
type RoomDeletedNotification struct {
	RoomID string `json:"room_id"`
}

// KickedNotification notifies a user they've been kicked
type KickedNotification struct {
	RoomID string `json:"room_id"`
	Reason string `json:"reason,omitempty"`
}

// ServerShutdownNotification notifies clients that the server is shutting down
type ServerShutdownNotification struct {
	Message     string `json:"message"`
	ShutdownIn  int    `json:"shutdown_in_seconds"` // Seconds until server shutdown
	RestartInfo string `json:"restart_info,omitempty"`
}

// GetUserRoomsRequest represents a request to get all rooms a user has joined
type GetUserRoomsRequest struct {
	BaseRequest
}

// UserRoomInfo represents information about a room a user has joined
type UserRoomInfo struct {
	RoomID        string    `json:"room_id"`
	RoomName      string    `json:"room_name"`
	IsConnected   bool      `json:"is_connected"`
	JoinedAt      time.Time `json:"joined_at"`
	LastConnected time.Time `json:"last_connected"`
}

// UserRoomsResponse represents a response containing all rooms a user has joined
type UserRoomsResponse struct {
	Rooms []UserRoomInfo `json:"rooms"`
}

// Helper functions

// GenerateMessageID gera um ID aleatório em formato hexadecimal com base no comprimento especificado
func GenerateMessageID() (string, error) {
	return GenerateRandomID(8)
}

func GenerateRoomID() string {
	id, err := GenerateRandomID(16)

	if err != nil {
		// Fall back to a timestamp-based ID if random generation fails
		return fmt.Sprintf("%06x", time.Now().UnixNano()%0xFFFFFF)
	}

	return id
}

// GenerateRandomID gera um ID aleatório em formato hexadecimal com o comprimento desejado
// Logic: Generate cryptographically secure random bytes and convert to hexadecimal format
func GenerateRandomID(length int) (string, error) {
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

// PasswordRegex returns a compiled regex for the default password pattern
func PasswordRegex() (*regexp.Regexp, error) {
	return regexp.Compile(DefaultPasswordPattern)
}

// ValidatePassword checks if a password matches the default password pattern
func ValidatePassword(password string) bool {
	regex, err := PasswordRegex()
	if err != nil {
		return false
	}
	return regex.MatchString(password)
}
