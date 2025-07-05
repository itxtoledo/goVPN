package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/storage"
	"github.com/itxtoledo/govpn/libs/models"
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
	VirtualNetwork    NetworkInterface
	SignalingServer   *SignalingClient
	RoomID            string
	RoomPassword      string
	Computers         []Computer
	connectionState   ConnectionState
	ReconnectAttempts int
	MaxReconnects     int

	// Dependencies
	RealtimeData       *data.RealtimeDataLayer
	ConfigManager      *ConfigManager
	refreshNetworkList func()
	refreshUI          func()
}

// NewNetworkManager creates a new instance of NetworkManager
func NewNetworkManager(realtimeData *data.RealtimeDataLayer, configManager *ConfigManager, refreshNetworkList func(), refreshUI func()) *NetworkManager {
	nm := &NetworkManager{
		connectionState:    ConnectionStateDisconnected,
		ReconnectAttempts:  0,
		MaxReconnects:      5,
		RealtimeData:       realtimeData,
		ConfigManager:      configManager,
		refreshNetworkList: refreshNetworkList,
		refreshUI:          refreshUI,
	}

	return nm
}

// Connect connects to the VPN network
func (nm *NetworkManager) Connect(serverAddress string) error {
	// Set state to connecting
	nm.connectionState = ConnectionStateConnecting
	// Update data layer
	nm.RealtimeData.SetConnectionState(data.StateConnecting)
	nm.RealtimeData.SetStatusMessage("Connecting...")

	// Update UI
	nm.refreshUI()

	// Initialize signaling server
	// Get public key from ConfigManager
	publicKey, _ := nm.ConfigManager.GetKeyPair()

	// Create a handler function for signaling client messages
	signalingHandler := func(messageType models.MessageType, payload []byte) {
		switch messageType {
		case models.TypeError:
			var errorPayload map[string]string
			if err := json.Unmarshal(payload, &errorPayload); err == nil {
				if errorMsg, ok := errorPayload["error"]; ok {
					log.Printf("Server error: %s", errorMsg)
					nm.RealtimeData.EmitEvent(data.EventError, errorMsg, nil)
				}
			}
		case models.TypeRoomDisconnected:
			nm.refreshNetworkList()
		case models.TypeRoomJoined:
			nm.refreshNetworkList()
		case models.TypeRoomCreated:
			nm.refreshNetworkList()
		case models.TypeLeaveRoom:
			nm.refreshNetworkList()
		case models.TypeKicked:
			nm.refreshNetworkList()
		case models.TypePeerJoined:
			nm.refreshNetworkList()
		case models.TypePeerLeft:
			nm.refreshNetworkList()
		case models.TypeUserRooms:
			// For TypeUserRooms, we need to unmarshal the payload to update the rooms list
			var userRoomsResponse models.UserRoomsResponse
			if err := json.Unmarshal(payload, &userRoomsResponse); err != nil {
				log.Printf("Failed to unmarshal user rooms response in handler: %v", err)
				return
			}

			// Convert models.Room to storage.Room
			updatedRooms := make([]*storage.Room, 0, len(userRoomsResponse.Rooms))
			for _, room := range userRoomsResponse.Rooms {
				storageRoom := &storage.Room{
					ID:            room.RoomID,
					Name:          room.RoomName,
					Password:      "", // Password is not received from server
					LastConnected: room.LastConnected,
				}
				updatedRooms = append(updatedRooms, storageRoom)
			}
			// Update the RealtimeDataLayer with the new rooms list
			nm.RealtimeData.SetRooms(updatedRooms)
			nm.refreshNetworkList()
		}
	}
	nm.SignalingServer = NewSignalingClient(publicKey, signalingHandler)

	// Connect to signaling server
	err := nm.SignalingServer.Connect(serverAddress)
	if err != nil {
		nm.connectionState = ConnectionStateDisconnected
		nm.RealtimeData.SetConnectionState(data.StateDisconnected)
		nm.RealtimeData.SetStatusMessage("Connection failed")
		return fmt.Errorf("failed to connect to signaling server: %v", err)
	}

	// Set state to connected
	nm.connectionState = ConnectionStateConnected
	nm.RealtimeData.SetConnectionState(data.StateConnected)
	nm.RealtimeData.SetStatusMessage("Connected")

	// O servidor já envia automaticamente a lista de salas do usuário ao conectar,
	// então não precisamos chamar GetUserRooms explicitamente
	log.Println("Aguardando lista de salas do servidor...")

	// Get room list
	nm.refreshNetworkList()

	return nil
}

// handleDisconnection handles disconnection from the server
func (nm *NetworkManager) handleDisconnection() {
	if nm.connectionState != ConnectionStateDisconnected {
		if nm.ReconnectAttempts < nm.MaxReconnects {
			nm.ReconnectAttempts++
			log.Printf("Disconnected from server, attempting to reconnect (%d/%d)", nm.ReconnectAttempts, nm.MaxReconnects)

			// Set state to connecting
			nm.connectionState = ConnectionStateConnecting
			nm.RealtimeData.SetConnectionState(data.StateConnecting)
			nm.RealtimeData.SetStatusMessage(fmt.Sprintf("Reconnecting (%d/%d)...", nm.ReconnectAttempts, nm.MaxReconnects))
			nm.refreshUI()

			// Try to reconnect
			err := nm.Connect(nm.SignalingServer.ServerAddress)
			if err != nil {
				log.Printf("Failed to reconnect: %v", err)
				// Set state to disconnected
				nm.connectionState = ConnectionStateDisconnected
				nm.RealtimeData.SetConnectionState(data.StateDisconnected)
				nm.RealtimeData.SetStatusMessage("Disconnected")
				nm.refreshUI()
			} else {
				// Successfully reconnected
				nm.ReconnectAttempts = 0
			}
		} else {
			log.Printf("Max reconnect attempts reached, giving up")
			// Set state to disconnected
			nm.connectionState = ConnectionStateDisconnected
			nm.RealtimeData.SetConnectionState(data.StateDisconnected)
			nm.RealtimeData.SetStatusMessage("Connection lost")
			nm.refreshUI()
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

	// Add the room to the RealtimeDataLayer's in-memory room list
	nm.RealtimeData.AddRoom(room)

	// Store room information for current connection
	nm.RoomID = res.RoomID
	nm.RoomPassword = password

	// Update data layer
	nm.RealtimeData.SetRoomInfo(res.RoomID, password)
	nm.RealtimeData.EmitEvent(data.EventRoomJoined, res.RoomID, nil)

	// Refresh network list now that we have added the room to memory
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

	return nil
}

// JoinRoom joins a room
func (nm *NetworkManager) JoinRoom(roomID string, password string, username string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Join room
	res, err := nm.SignalingServer.JoinRoom(roomID, password, username)
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
	for i, existingRoom := range nm.RealtimeData.GetRooms() {
		if existingRoom.ID == roomID {
			// Update existing room
			nm.RealtimeData.UpdateRoom(i, room)
			roomExists = true
			break
		}
	}

	// If room doesn't exist, add it
	if !roomExists {
		nm.RealtimeData.AddRoom(room)
	}

	log.Printf("Room joined: ID=%s, Name=%s", roomID, roomName)

	// Store room information for current connection
	nm.RoomID = roomID
	nm.RoomPassword = password

	// Update data layer
	nm.RealtimeData.SetRoomInfo(roomID, password)
	nm.RealtimeData.EmitEvent(data.EventRoomJoined, roomID, nil)

	// Refresh network list now that we have added the room to memory
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

	return nil
}

// ConnectRoom connects to a previously joined room
func (nm *NetworkManager) ConnectRoom(roomID string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Connect to room
	res, err := nm.SignalingServer.ConnectRoom(roomID, "User")
	if err != nil {
		return fmt.Errorf("failed to connect to room: %v", err)
	}

	// Use roomName from the response
	roomName := res.RoomName

	// Find the room in memory to get the stored password
	var roomPassword string
	roomExists := false
	for i, existingRoom := range nm.RealtimeData.GetRooms() {
		if existingRoom.ID == roomID {
			roomPassword = existingRoom.Password
			roomExists = true

			// Update lastConnected time
			existingRoom.LastConnected = time.Now()
			nm.RealtimeData.UpdateRoom(i, existingRoom)
			break
		}
	}

	if !roomExists {
		return fmt.Errorf("room not found in local storage")
	}

	log.Printf("Room connected: ID=%s, Name=%s", roomID, roomName)

	// Store room information for current connection
	nm.RoomID = roomID
	nm.RoomPassword = roomPassword

	// Update data layer
	nm.RealtimeData.SetRoomInfo(roomID, roomPassword)
	nm.RealtimeData.EmitEvent(data.EventRoomJoined, roomID, nil)

	// Refresh network list now that we have re-connected to the room
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

	return nil
}

// DisconnectRoom disconnects from a room without leaving it
func (nm *NetworkManager) DisconnectRoom(roomID string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	log.Printf("Disconnecting from room with ID: %s", roomID)

	// Disconnect from room
	_, err := nm.SignalingServer.DisconnectRoom(roomID)
	if err != nil {
		log.Printf("Error from SignalingClient when disconnecting from room: %v", err)
		return fmt.Errorf("failed to disconnect from room: %v", err)
	}

	// If we're disconnecting from the current room, clear our room information
	if nm.RoomID == roomID {
		nm.RoomID = ""
		nm.RoomPassword = ""

		// Update data layer
		nm.RealtimeData.SetRoomInfo("Not connected", "")
		nm.RealtimeData.EmitEvent(data.EventRoomDisconnected, roomID, nil)
	}

	// Clear the computers list
	nm.Computers = []Computer{}

	// Refresh the network list UI
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

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
	nm.RealtimeData.RemoveRoom(roomID)

	// Clear room information
	nm.RoomID = ""
	nm.RoomPassword = ""

	// Update data layer
	nm.RealtimeData.SetRoomInfo("Not connected", "")
	nm.RealtimeData.EmitEvent(data.EventRoomLeft, roomID, nil)

	// Refresh the network list UI
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

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
	nm.RealtimeData.RemoveRoom(roomID)

	// If we're leaving the current room, clear our room information
	if nm.RoomID == roomID {
		nm.RoomID = ""
		nm.RoomPassword = ""

		// Update data layer
		nm.RealtimeData.SetRoomInfo("Not connected", "")
	}

	// Emit the room left event regardless
	nm.RealtimeData.EmitEvent(data.EventRoomLeft, roomID, nil)

	// Refresh the network list UI
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

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
		nm.RealtimeData.SetRoomInfo("Not connected", "")
	}

	// Remove room from memory
	nm.RealtimeData.RemoveRoom(roomID)

	// Emit the event
	nm.RealtimeData.EmitEvent(data.EventRoomDeleted, roomID, nil)

	// Refresh the network list UI
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

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
	nm.RealtimeData.SetConnectionState(data.StateDisconnected)
	nm.RealtimeData.SetStatusMessage("Disconnected")
	nm.ReconnectAttempts = 0

	// Update UI
	nm.refreshUI()

	return nil
}
