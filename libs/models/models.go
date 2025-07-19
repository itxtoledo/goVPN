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
	TypeCreateNetwork       MessageType = "CreateNetwork"
	TypeJoinNetwork         MessageType = "JoinNetwork"
	TypeConnectNetwork      MessageType = "ConnectNetwork"
	TypeDisconnectNetwork   MessageType = "DisconnectNetwork"
	TypeLeaveNetwork        MessageType = "LeaveNetwork"
	TypeKick                MessageType = "Kick"
	TypeRename              MessageType = "Rename"
	TypePing                MessageType = "Ping"
	TypeGetComputerNetworks MessageType = "GetComputerNetworks"

	// Server to client message types
	TypeError                MessageType = "Error"
	TypeNetworkCreated       MessageType = "NetworkCreated"
	TypeNetworkJoined        MessageType = "NetworkJoined"
	TypeNetworkConnected     MessageType = "NetworkConnected"
	TypeNetworkDisconnected  MessageType = "NetworkDisconnected"
	TypeNetworkDeleted       MessageType = "NetworkDeleted"
	TypeNetworkRenamed       MessageType = "NetworkRenamed"
	TypeComputerJoined       MessageType = "ComputerJoined"
	TypeComputerLeft         MessageType = "ComputerLeft"
	TypeComputerConnected    MessageType = "ComputerConnected"
	TypeComputerDisconnected MessageType = "ComputerDisconnected"
	TypeKicked               MessageType = "Kicked"
	TypeKickSuccess          MessageType = "KickSuccess"
	TypeRenameSuccess        MessageType = "RenameSuccess"
	TypeDeleteSuccess        MessageType = "DeleteSuccess"
	TypeServerShutdown       MessageType = "ServerShutdown"
	TypeComputerNetworks     MessageType = "ComputerNetworks"
	TypeClientIPInfo         MessageType = "ClientIPInfo"
)

// Password validation constants
const (
	// DefaultPasswordPattern is the default password validation pattern: exactly 4 numeric digits
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

// Network represents a network or network
type Network struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Password    string `json:"password,omitempty"`
	ClientCount int    `json:"client_count"`
}

// Event-specific request structs

// CreateNetworkRequest represents a request to create a new network
type CreateNetworkRequest struct {
	BaseRequest
	NetworkName string `json:"network_name"`
	Password    string `json:"password"`
}

// CreateNetworkResponse represents a response to a network creation request
type CreateNetworkResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	Password    string `json:"password"`
	PublicKey   string `json:"public_key"`
	PeerIP      string `json:"peer_ip"`
}

// JoinNetworkRequest represents a request to join an existing network
type JoinNetworkRequest struct {
	BaseRequest
	NetworkID    string `json:"network_id"`
	Password     string `json:"password"`
	ComputerName string `json:"computername,omitempty"`
}

// JoinNetworkResponse represents a response to a network join request
type JoinNetworkResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	PeerIP      string `json:"peer_ip"`
}

// ConnectNetworkRequest represents a request to connect to a previously joined network
type ConnectNetworkRequest struct {
	BaseRequest
	NetworkID    string `json:"network_id"`
	ComputerName string `json:"computername,omitempty"`
}

// ConnectNetworkResponse represents a response to a network connection request
type ConnectNetworkResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	PeerIP      string `json:"peer_ip"`
}

// DisconnectNetworkRequest represents a request to disconnect from a network (but stay joined)
type DisconnectNetworkRequest struct {
	BaseRequest
	NetworkID string `json:"network_id"`
}

// DisconnectNetworkResponse represents a response to a network disconnect request
type DisconnectNetworkResponse struct {
	NetworkID string `json:"network_id"`
}

// LeaveNetworkRequest represents a request to leave a network
type LeaveNetworkRequest struct {
	BaseRequest
	NetworkID string `json:"network_id"`
}

// LeaveNetworkResponse confirms a client has left a network
type LeaveNetworkResponse struct {
	NetworkID string `json:"network_id"`
}

// Network management structs

// KickRequest represents a request to kick a computer from a network
type KickRequest struct {
	BaseRequest
	NetworkID string `json:"network_id"`
	TargetID  string `json:"target_id"`
}

// KickResponse confirms a computer has been kicked
type KickResponse struct {
	NetworkID string `json:"network_id"`
	TargetID  string `json:"target_id"`
}

// RenameRequest represents a request to rename a network
type RenameRequest struct {
	BaseRequest
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
}

// RenameResponse confirms a network has been renamed
type RenameResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
}

// Computer notification structs

// ComputerJoinedNotification notifies that a computer has joined the network
type ComputerJoinedNotification struct {
	NetworkID    string `json:"network_id"`
	PublicKey    string `json:"public_key"`
	ComputerName string `json:"computername,omitempty"`
	PeerIP       string `json:"peer_ip,omitempty"`
}

// ComputerLeftNotification notifies that a computer has left the network
type ComputerLeftNotification struct {
	NetworkID string `json:"network_id"`
	PublicKey string `json:"public_key"`
}

// ComputerConnectedNotification notifies that a computer has connected to the network
type ComputerConnectedNotification struct {
	NetworkID    string `json:"network_id"`
	PublicKey    string `json:"public_key"`
	ComputerName string `json:"computername,omitempty"`
	PeerIP       string `json:"peer_ip,omitempty"`
}

// ComputerDisconnectedNotification notifies that a computer has disconnected from the network (but not left)
type ComputerDisconnectedNotification struct {
	NetworkID string `json:"network_id"`
	PublicKey string `json:"public_key"`
}

// NetworkDeletedNotification notifies that a network has been deleted
type NetworkDeletedNotification struct {
	NetworkID string `json:"network_id"`
}

// KickedNotification notifies a computer they've been kicked
type KickedNotification struct {
	NetworkID string `json:"network_id"`
	Reason    string `json:"reason,omitempty"`
}

// ServerShutdownNotification notifies clients that the server is shutting down
type ServerShutdownNotification struct {
	Message     string `json:"message"`
	ShutdownIn  int    `json:"shutdown_in_seconds"` // Seconds until server shutdown
	RestartInfo string `json:"restart_info,omitempty"`
}

// GetComputerNetworksRequest represents a request to get all networks a computer has joined
type GetComputerNetworksRequest struct {
	BaseRequest
}

// ComputerNetworkInfo represents information about a network a computer has joined
type ComputerNetworkInfo struct {
	NetworkID     string    `json:"network_id"`
	NetworkName   string    `json:"network_name"`
	JoinedAt      time.Time `json:"joined_at"`
	LastConnected time.Time `json:"last_connected"`
}

// ComputerNetworksResponse represents a response containing all networks a computer has joined
type ComputerNetworksResponse struct {
	Networks []ComputerNetworkInfo `json:"networks"`
}

// ClientIPInfoResponse represents client IP address information
type ClientIPInfoResponse struct {
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
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
