package main

import (
	"fmt"
	"log"
	"time"

	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// NetworkInterface define a interface mínima para a rede virtual
type NetworkInterface interface {
	GetLocalIP() string
	GetPeerCount() int
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

	// Set state to connected
	nm.connectionState = ConnectionStateConnected
	nm.UI.RealtimeData.SetConnectionState(data.StateConnected)
	nm.UI.RealtimeData.SetStatusMessage("Connected")

	// O servidor já envia automaticamente a lista de salas do usuário ao conectar,
	// então não precisamos chamar GetUserRooms explicitamente
	log.Println("Aguardando lista de salas do servidor...")

	// Get room list
	nm.UI.refreshNetworkList()

	return nil
}

// TODO check if we can delete this
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

	// Check if the room was successfully created and has a valid ID
	if res == nil || res.RoomID == "" {
		return fmt.Errorf("failed to create room: invalid server response")
	}

	log.Printf("Room created: ID=%s, Name=%s", res.RoomID, name)

	// Store room information in memory
	room := &storage.Room{
		ID:            res.RoomID,
		Name:          name,
		Password:      password,
		LastConnected: time.Now(),
	}

	// Add the room to the UI's in-memory room list
	// Use append to add the new room to the existing slice
	nm.UI.Rooms = append(nm.UI.Rooms, room)

	// Store room information for current connection
	nm.RoomID = res.RoomID
	nm.RoomPassword = password

	// Update data layer
	nm.UI.RealtimeData.SetRoomInfo(res.RoomID, password)
	nm.UI.RealtimeData.EmitEvent(data.EventRoomJoined, res.RoomID, nil)

	// Refresh network list now that we have added the room to memory
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

	// Use roomName from the response
	roomName := res.RoomName

	// Store room information in memory
	room := &storage.Room{
		ID:            roomID,
		Name:          roomName,
		Password:      password,
		LastConnected: time.Now(),
	}

	// Check if the room already exists in memory
	roomExists := false
	for i, existingRoom := range nm.UI.Rooms {
		if existingRoom.ID == roomID {
			// Update existing room
			nm.UI.Rooms[i] = room
			roomExists = true
			break
		}
	}

	// If room doesn't exist, add it
	if !roomExists {
		nm.UI.Rooms = append(nm.UI.Rooms, room)
	}

	log.Printf("Room joined: ID=%s, Name=%s", roomID, roomName)

	// Store room information for current connection
	nm.RoomID = roomID
	nm.RoomPassword = password

	// Update data layer
	nm.UI.RealtimeData.SetRoomInfo(roomID, password)
	nm.UI.RealtimeData.EmitEvent(data.EventRoomJoined, roomID, nil)

	// Refresh network list now that we have added the room to memory
	nm.UI.refreshNetworkList()

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

	// Remove the room from memory
	for i, room := range nm.UI.Rooms {
		if room.ID == roomID {
			// Remove this room from the slice
			nm.UI.Rooms = append(nm.UI.Rooms[:i], nm.UI.Rooms[i+1:]...)
			break
		}
	}

	// Clear room information
	nm.RoomID = ""
	nm.RoomPassword = ""

	// Update data layer
	nm.UI.RealtimeData.SetRoomInfo("Not connected", "")
	nm.UI.RealtimeData.EmitEvent(data.EventRoomLeft, roomID, nil)

	// Refresh the network list UI
	nm.UI.refreshNetworkList()

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

	// Remove room from memory
	for i, room := range nm.UI.Rooms {
		if room.ID == roomID {
			// Remove this room from the slice
			nm.UI.Rooms = append(nm.UI.Rooms[:i], nm.UI.Rooms[i+1:]...)
			break
		}
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

	// Refresh the network list UI
	nm.UI.refreshNetworkList()

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
	}

	// Remove room from memory
	for i, room := range nm.UI.Rooms {
		if room.ID == roomID {
			// Remove this room from the slice
			nm.UI.Rooms = append(nm.UI.Rooms[:i], nm.UI.Rooms[i+1:]...)
			break
		}
	}

	// Emit the event
	nm.UI.RealtimeData.EmitEvent(data.EventRoomDeleted, roomID, nil)

	// Refresh the network list UI
	nm.UI.refreshNetworkList()

	// Update UI
	nm.UI.refreshUI()

	return nil
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
