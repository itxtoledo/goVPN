package models

import "time"

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

// Event-specific request structs

// CreateNetworkRequest represents a request to create a new network
type CreateNetworkRequest struct {
	BaseRequest
	NetworkName string `json:"network_name"`
	PIN         string `json:"pin"`
}

// CreateNetworkResponse represents a response to a network creation request
type CreateNetworkResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	PIN         string `json:"pin"`
	PublicKey   string `json:"public_key"`
	ComputerIP  string `json:"computer_ip"`
}

// JoinNetworkRequest represents a request to join an existing network
type JoinNetworkRequest struct {
	BaseRequest
	NetworkID    string `json:"network_id"`
	PIN          string `json:"pin"`
	ComputerName string `json:"computername,omitempty"`
}

// JoinNetworkResponse represents a response to a network join request
type JoinNetworkResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	ComputerIP  string `json:"computer_ip"`
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
	ComputerIP  string `json:"computer_ip"`
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
	ComputerIP   string `json:"computer_ip,omitempty"`
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
	ComputerIP   string `json:"computer_ip,omitempty"`
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

// ComputerInfo represents information about a computer in a network
type ComputerInfo struct {
	Name       string `json:"name"`
	ComputerIP string `json:"computer_ip"`
	IsAdmin    bool   `json:"is_admin"`
	PublicKey  string `json:"public_key"`
}

// ComputerNetworkInfo represents information about a network a computer has joined
type ComputerNetworkInfo struct {
	NetworkID     string         `json:"network_id"`
	NetworkName   string         `json:"network_name"`
	JoinedAt      time.Time      `json:"joined_at"`
	LastConnected time.Time      `json:"last_connected"`
	ComputerIP    string         `json:"computer_ip,omitempty"`
	Computers     []ComputerInfo `json:"computers"`
}

// ComputerNetworksResponse represents a response containing all networks a computer has joined
type ComputerNetworksResponse struct {
	Networks []ComputerNetworkInfo `json:"networks"`
}

// Computer represents a computer connected to a network
type Computer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	OwnerID  string `json:"owner_id"`
	IsOnline bool   `json:"is_online"`
	PeerIP   string `json:"computer_ip,omitempty"`
}
