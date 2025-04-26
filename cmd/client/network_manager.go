package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/itxtoledo/govpn/cmd/client/data"
)

// NetworkInterface define a interface mínima para a rede virtual
type NetworkInterface interface {
	GetLocalIP() string
	GetPeerCount() int
	GetAverageLatency() float64
	GetBytesSent() int64
	GetBytesReceived() int64
}

// Room represents a VPN room
type Room struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Password    string `json:"password"`
	OwnerID     string `json:"owner_id"`
	CreatedAt   string `json:"created_at"`
}

// Computer represents a computer in the VPN network
type Computer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	OwnerID  string `json:"owner_id"`
	IsOnline bool   `json:"is_online"`
}

// ConnectionState represents the state of the connection
type ConnectionState int

const (
	ConnectionStateDisconnected ConnectionState = iota
	ConnectionStateConnecting
	ConnectionStateConnected
)

// NetworkManager handles the VPN network
type NetworkManager struct {
	UI                *UIManager
	VirtualNetwork    NetworkInterface
	SignalingServer   *SignalingClient
	RoomName          string
	RoomPassword      string
	RoomDescription   string
	Computers         []Computer
	connectionState   ConnectionState
	ReconnectAttempts int
	MaxReconnects     int
}

// NewNetworkManager creates a new instance of NetworkManager
func NewNetworkManager(ui *UIManager) *NetworkManager {
	nm := &NetworkManager{
		UI:                ui,
		connectionState:   ConnectionStateDisconnected,
		ReconnectAttempts: 0,
		MaxReconnects:     5,
	}

	return nm
}

// Connect connects to the VPN network
func (nm *NetworkManager) Connect(serverAddress string) error {
	// Set state to connecting
	nm.connectionState = ConnectionStateConnecting
	// Update data layer
	nm.UI.RealtimeData.SetConnectionState(data.StateConnecting)
	nm.UI.RealtimeData.SetStatusMessage("Connecting...")

	// Update UI
	nm.UI.refreshUI()

	// Initialize signaling server
	nm.SignalingServer = NewSignalingClient(nm.UI)

	// Connect to signaling server
	err := nm.SignalingServer.Connect(serverAddress)
	if err != nil {
		nm.connectionState = ConnectionStateDisconnected
		nm.UI.RealtimeData.SetConnectionState(data.StateDisconnected)
		nm.UI.RealtimeData.SetStatusMessage("Connection failed")
		return fmt.Errorf("failed to connect to signaling server: %v", err)
	}

	// Initialize virtual network
	// A inicialização real da rede é feita em outro lugar
	// Para este exemplo, usamos uma implementação simulada

	// Set state to connected (simulado para este exemplo)
	nm.connectionState = ConnectionStateConnected
	nm.UI.RealtimeData.SetConnectionState(data.StateConnected)
	nm.UI.RealtimeData.SetStatusMessage("Connected")

	// Get room list
	nm.UI.refreshNetworkList()

	// Start heartbeat
	go nm.heartbeat()

	return nil
}

// heartbeat sends heartbeat to the server
func (nm *NetworkManager) heartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if nm.connectionState != ConnectionStateConnected {
				return
			}

			// Get network stats
			var peerCount int
			var latency float64
			var sentBytes float64
			var receivedBytes float64

			if nm.VirtualNetwork != nil {
				// Calcular estatísticas da rede
				peerCount = nm.VirtualNetwork.GetPeerCount()
				latency = nm.VirtualNetwork.GetAverageLatency()
				sentBytes = float64(nm.VirtualNetwork.GetBytesSent())
				receivedBytes = float64(nm.VirtualNetwork.GetBytesReceived())
			}

			// Update data layer with network stats
			nm.UI.RealtimeData.UpdateNetworkStats(peerCount, latency, sentBytes, receivedBytes)

			// Update IP info in case it changed
			if nm.VirtualNetwork != nil {
				nm.UI.RealtimeData.SetLocalIP(nm.VirtualNetwork.GetLocalIP())
			}

			// Send heartbeat
			err := nm.SignalingServer.SendHeartbeat()
			if err != nil {
				log.Printf("Error sending heartbeat: %v", err)
				nm.handleDisconnection()
				return
			}
		}
	}
}

// handleDisconnection handles disconnection from the server
func (nm *NetworkManager) handleDisconnection() {
	if nm.connectionState != ConnectionStateDisconnected {
		if nm.ReconnectAttempts < nm.MaxReconnects {
			nm.ReconnectAttempts++
			log.Printf("Disconnected from server, attempting to reconnect (%d/%d)", nm.ReconnectAttempts, nm.MaxReconnects)

			// Set state to connecting
			nm.connectionState = ConnectionStateConnecting
			nm.UI.RealtimeData.SetConnectionState(data.StateConnecting)
			nm.UI.RealtimeData.SetStatusMessage(fmt.Sprintf("Reconnecting (%d/%d)...", nm.ReconnectAttempts, nm.MaxReconnects))
			nm.UI.refreshUI()

			// Try to reconnect
			err := nm.Connect(nm.SignalingServer.ServerAddress)
			if err != nil {
				log.Printf("Failed to reconnect: %v", err)
				// Set state to disconnected
				nm.connectionState = ConnectionStateDisconnected
				nm.UI.RealtimeData.SetConnectionState(data.StateDisconnected)
				nm.UI.RealtimeData.SetStatusMessage("Disconnected")
				nm.UI.refreshUI()
			} else {
				// Successfully reconnected
				nm.ReconnectAttempts = 0
			}
		} else {
			log.Printf("Max reconnect attempts reached, giving up")
			// Set state to disconnected
			nm.connectionState = ConnectionStateDisconnected
			nm.UI.RealtimeData.SetConnectionState(data.StateDisconnected)
			nm.UI.RealtimeData.SetStatusMessage("Connection lost")
			nm.UI.refreshUI()
		}
	}
}

// GetConnectionState returns the connection state
func (nm *NetworkManager) GetConnectionState() ConnectionState {
	return nm.connectionState
}

// CreateRoom creates a new room
func (nm *NetworkManager) CreateRoom(name string, description string, password string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Create room
	err := nm.SignalingServer.CreateRoom(name, description, password)
	if err != nil {
		return fmt.Errorf("failed to create room: %v", err)
	}

	// Get room list
	nm.UI.refreshNetworkList()

	return nil
}

// JoinRoom joins a room
func (nm *NetworkManager) JoinRoom(roomName string, password string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Join room
	err := nm.SignalingServer.JoinRoom(roomName, password)
	if err != nil {
		return fmt.Errorf("failed to join room: %v", err)
	}

	// Get details for the room from the cached list
	var description string
	for _, room := range nm.UI.Rooms {
		if room.Name == roomName {
			description = room.Description
			break
		}
	}

	// Store room information
	nm.RoomName = roomName
	nm.RoomPassword = password
	nm.RoomDescription = description

	// Update data layer
	nm.UI.RealtimeData.SetRoomInfo(roomName, password, description)
	nm.UI.RealtimeData.EmitEvent(data.EventRoomJoined, roomName, nil)

	// Update UI
	nm.UI.refreshUI()

	return nil
}

// LeaveRoom leaves the current room
func (nm *NetworkManager) LeaveRoom() error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Leave room
	err := nm.SignalingServer.LeaveRoom(nm.RoomName)
	if err != nil {
		return fmt.Errorf("failed to leave room: %v", err)
	}

	// Clear room information
	roomName := nm.RoomName
	nm.RoomName = ""
	nm.RoomPassword = ""
	nm.RoomDescription = ""

	// Update data layer
	nm.UI.RealtimeData.SetRoomInfo("Not connected", "", "")
	nm.UI.RealtimeData.EmitEvent(data.EventRoomLeft, roomName, nil)

	// Update UI
	nm.UI.refreshUI()

	return nil
}

// GetRoomList gets the list of rooms
func (nm *NetworkManager) GetRoomList() ([]*Room, error) {
	if nm.connectionState != ConnectionStateConnected {
		return nil, fmt.Errorf("not connected to server")
	}

	// Get room list
	roomsData, err := nm.SignalingServer.GetRoomList()
	if err != nil {
		return nil, fmt.Errorf("failed to get room list: %v", err)
	}

	// Parse room list
	var rooms []*Room
	err = json.Unmarshal([]byte(roomsData), &rooms)
	if err != nil {
		return nil, fmt.Errorf("failed to parse room list: %v", err)
	}

	return rooms, nil
}

// Disconnect disconnects from the VPN network
func (nm *NetworkManager) Disconnect() error {
	if nm.connectionState == ConnectionStateDisconnected {
		return nil
	}

	// Leave room if in a room
	if nm.RoomName != "" {
		err := nm.LeaveRoom()
		if err != nil {
			log.Printf("Error leaving room: %v", err)
		}
	}

	// Disconnect from signaling server
	if nm.SignalingServer != nil {
		err := nm.SignalingServer.Disconnect()
		if err != nil {
			log.Printf("Error disconnecting from signaling server: %v", err)
		}
	}

	// Stop virtual network
	if nm.VirtualNetwork != nil {
		nm.VirtualNetwork = nil
	}

	// Set state to disconnected
	nm.connectionState = ConnectionStateDisconnected
	nm.UI.RealtimeData.SetConnectionState(data.StateDisconnected)
	nm.UI.RealtimeData.SetStatusMessage("Disconnected")
	nm.ReconnectAttempts = 0

	// Update UI
	nm.UI.refreshUI()

	return nil
}
