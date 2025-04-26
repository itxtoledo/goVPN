package main

import (
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
	RoomID            string
	RoomPassword      string
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
	// Get public key from UI's realtime data layer
	publicKey := nm.UI.VPN.PublicKeyStr
	nm.SignalingServer = NewSignalingClient(nm.UI, publicKey)

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
func (nm *NetworkManager) CreateRoom(name string, password string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Create room
	res, err := nm.SignalingServer.CreateRoom(name, password)
	if err != nil {
		return fmt.Errorf("failed to create room: %v", err)
	}

	// Save room to local database
	err = nm.UI.VPN.DBManager.SaveRoom(res.RoomID, name, password)
	if err != nil {
		log.Printf("Warning: Could not save room to database: %v", err)
		// Continue even if database save fails
	} else {
		log.Printf("Room saved to database: ID=%s, Name=%s", res.RoomID, name)
	}

	// Update room connection time
	err = nm.UI.VPN.DBManager.UpdateRoomConnection(res.RoomID)
	if err != nil {
		log.Printf("Warning: Could not update room connection time: %v", err)
		// Continue even if update fails
	}

	// Store room information
	nm.RoomID = res.RoomID
	nm.RoomPassword = password

	// Update data layer
	nm.UI.RealtimeData.SetRoomInfo(res.RoomID, password)
	nm.UI.RealtimeData.EmitEvent(data.EventRoomJoined, res.RoomID, nil)

	// Get room list
	nm.UI.refreshNetworkList()

	// Update UI
	nm.UI.refreshUI()

	return nil
}

// JoinRoom joins a room
func (nm *NetworkManager) JoinRoom(roomID string, password string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Join room
	res, err := nm.SignalingServer.JoinRoom(roomID, password)
	if err != nil {
		return fmt.Errorf("failed to join room: %v", err)
	}

	// Use "Sala " + roomID as the room name since there's no GetRoomName method on the backend
	roomName := res.RoomName

	// Save room to local database
	err = nm.UI.VPN.DBManager.SaveRoom(roomID, roomName, password)
	if err != nil {
		log.Printf("Warning: Could not save room to database: %v", err)
		// Continue even if database save fails
	} else {
		log.Printf("Room saved to database: ID=%s, Name=%s", roomID, roomName)
	}

	// Update room connection time
	err = nm.UI.VPN.DBManager.UpdateRoomConnection(roomID)
	if err != nil {
		log.Printf("Warning: Could not update room connection time: %v", err)
		// Continue even if update fails
	}

	// Store room information
	nm.RoomID = roomID
	nm.RoomPassword = password

	// Update data layer
	nm.UI.RealtimeData.SetRoomInfo(roomID, password)
	nm.UI.RealtimeData.EmitEvent(data.EventRoomJoined, roomID, nil)

	// Update UI
	nm.UI.refreshUI()

	return nil
}

// LeaveRoom leaves the current room
func (nm *NetworkManager) LeaveRoom() error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	log.Printf("Leaving room with ID: %s", nm.RoomID)

	// Store room ID before clearing
	roomID := nm.RoomID

	// Leave room
	_, err := nm.SignalingServer.LeaveRoom(roomID)
	if err != nil {
		log.Printf("Error from SignalingClient when leaving room: %v", err)
		return fmt.Errorf("failed to leave room: %v", err)
	}

	// Delete the room from the local database
	err = nm.UI.VPN.DBManager.DeleteRoom(roomID)
	if err != nil {
		log.Printf("Error deleting room from database: %v", err)
		// Continue even if database deletion fails
	}

	// Clear room information
	nm.RoomID = ""
	nm.RoomPassword = ""

	// Update data layer
	nm.UI.RealtimeData.SetRoomInfo("Not connected", "")
	nm.UI.RealtimeData.EmitEvent(data.EventRoomLeft, roomID, nil)

	// Update UI
	nm.UI.refreshUI()

	return nil
}

// LeaveRoomById leaves a specific room by ID
func (nm *NetworkManager) LeaveRoomById(roomID string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	log.Printf("Leaving room with ID: %s", roomID)

	// Leave room
	_, err := nm.SignalingServer.LeaveRoom(roomID)
	if err != nil {
		log.Printf("Error from SignalingClient when leaving room: %v", err)
		return fmt.Errorf("failed to leave room: %v", err)
	}

	// Delete the room from the local database
	err = nm.UI.VPN.DBManager.DeleteRoom(roomID)
	if err != nil {
		log.Printf("Error deleting room from database: %v", err)
		// Continue even if database deletion fails
	}

	// If we're leaving the current room, clear our room information
	if nm.RoomID == roomID {
		nm.RoomID = ""
		nm.RoomPassword = ""

		// Update data layer
		nm.UI.RealtimeData.SetRoomInfo("Not connected", "")
	}

	// Emit the room left event regardless
	nm.UI.RealtimeData.EmitEvent(data.EventRoomLeft, roomID, nil)

	// Update UI
	nm.UI.refreshUI()

	return nil
}

// HandleRoomDeleted handles when a room has been deleted
func (nm *NetworkManager) HandleRoomDeleted(roomID string) error {
	log.Printf("Handling room deletion for ID: %s", roomID)

	// If we're in this room, clear our room data
	if nm.RoomID == roomID {
		nm.RoomID = ""
		nm.RoomPassword = ""

		// Update data layer
		nm.UI.RealtimeData.SetRoomInfo("Not connected", "")
		nm.UI.RealtimeData.EmitEvent(data.EventRoomDeleted, roomID, nil)

		// Update UI
		nm.UI.refreshUI()
	}

	// Delete the room from the database using the existing database manager
	return nm.UI.VPN.DBManager.DeleteRoom(roomID)
}

// Disconnect disconnects from the VPN network
func (nm *NetworkManager) Disconnect() error {
	if nm.connectionState == ConnectionStateDisconnected {
		return nil
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
