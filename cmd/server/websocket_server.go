// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/server/websocket_server.go
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/itxtoledo/govpn/cmd/server/logger"
	"github.com/itxtoledo/govpn/libs/crypto_utils"
	"github.com/itxtoledo/govpn/libs/models"
)

// ServerRoom extends the basic Room model with server-specific fields
type ServerRoom struct {
	models.Room                    // Embed the Room from models package
	PublicKey    ed25519.PublicKey `json:"-"`          // Not stored in Supabase directly
	PublicKeyB64 string            `json:"public_key"` // Stored as base64 string in Supabase
	CreatedAt    time.Time         `json:"created_at"`
	LastActive   time.Time         `json:"last_active"`
}

// SupabaseRoom is a struct for room data stored in Supabase
type SupabaseRoom struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Password   string    `json:"password"`
	PublicKey  string    `json:"public_key"` // Base64 encoded public key
	CreatedAt  time.Time `json:"created_at"`
	LastActive time.Time `json:"last_active"`
}

// WebSocketServer manages the WebSocket connections and room handling
type WebSocketServer struct {
	clients           map[*websocket.Conn]string   // Maps connection to roomID
	networks          map[string][]*websocket.Conn // Maps roomID to list of connections
	clientToPublicKey map[*websocket.Conn]string   // Maps connection to public key
	mu                sync.RWMutex
	config            Config
	supabaseManager   *SupabaseManager
	upgrader          websocket.Upgrader
	passwordRegex     *regexp.Regexp

	// Server statistics
	statsManager *StatsManager

	// Graceful shutdown
	shutdownChan chan struct{}
	httpServer   *http.Server
	isShutdown   bool
}

// NewWebSocketServer creates a new WebSocket server instance
// Logic: Initialize server with required configurations, establish Supabase connection,
// and prepare necessary maps for tracking clients and rooms
func NewWebSocketServer(cfg Config) (*WebSocketServer, error) {
	// Use diretamente o padrão de senha do módulo models
	passwordRegex, err := models.PasswordRegex()
	if err != nil {
		return nil, fmt.Errorf("falha ao compilar o padrão de senha: %w", err)
	}

	// Create Supabase manager
	supaMgr, err := NewSupabaseManager(cfg.SupabaseURL, cfg.SupabaseKey, cfg.SupabaseRoomsTable, cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create Supabase manager: %w", err)
	}

	// Configure WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return cfg.AllowAllOrigins
		},
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
	}

	// Create stats manager
	statsManager := NewStatsManager(cfg)

	return &WebSocketServer{
		clients:           make(map[*websocket.Conn]string),
		networks:          make(map[string][]*websocket.Conn),
		clientToPublicKey: make(map[*websocket.Conn]string),
		config:            cfg,
		supabaseManager:   supaMgr,
		upgrader:          upgrader,
		passwordRegex:     passwordRegex,
		statsManager:      statsManager,
		shutdownChan:      make(chan struct{}),
		httpServer:        &http.Server{},
		isShutdown:        false,
	}, nil
}

// handleWebSocketEndpoint is the HTTP handler for WebSocket connections
// Logic: Upgrade HTTP connection to WebSocket, then handle incoming messages in a loop
// until the connection is closed
func (s *WebSocketServer) HandleWebSocketEndpoint(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection", "error", err)
		return
	}
	defer conn.Close()

	s.statsManager.IncrementConnectionsTotal()
	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	// Extract client ID (public key) from header if available
	publicKeyHeader := r.Header.Get("X-Client-ID")
	if publicKeyHeader != "" && s.config.LogLevel == "debug" {
		logger.Debug("Client connected with public key", "publicKey", publicKeyHeader)

		// When client connects with public key, automatically send their rooms
		msgID, _ := models.GenerateMessageID()

		// Create and handle a request to get user rooms
		req := models.GetUserRoomsRequest{
			BaseRequest: models.BaseRequest{
				PublicKey: publicKeyHeader,
			},
		}

		// Send the user rooms in a separate goroutine to avoid blocking the connection setup
		go s.handleGetUserRooms(conn, req, msgID)
	}

	for {
		var sigMsg models.SignalingMessage
		err := conn.ReadJSON(&sigMsg)
		if err != nil {
			s.handleDisconnect(conn)
			return
		}

		s.statsManager.IncrementMessagesProcessed()

		// Salva o ID da mensagem original para incluir na resposta
		originalID := sigMsg.ID

		// Process the message based on its type
		switch sigMsg.Type {
		case models.TypeCreateRoom:
			var req models.CreateRoomRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid create room request format", originalID)
				continue
			}

			s.handleCreateRoom(conn, req, originalID)

		case models.TypeJoinRoom:
			var req models.JoinRoomRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid join room request format", originalID)
				continue
			}

			s.handleJoinRoom(conn, req, originalID)

		case models.TypeConnectRoom:
			var req models.ConnectRoomRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid connect room request format", originalID)
				continue
			}

			s.handleConnectRoom(conn, req, originalID)

		case models.TypeDisconnectRoom:
			var req models.DisconnectRoomRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid disconnect room request format", originalID)
				continue
			}

			s.handleDisconnectRoom(conn, req, originalID)

		case models.TypeLeaveRoom:
			var req models.LeaveRoomRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid leave room request format", originalID)
				continue
			}

			s.handleLeaveRoom(conn, req, originalID)

		case models.TypeKick:
			var req models.KickRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid kick request format", originalID)
				continue
			}

			s.handleKick(conn, req, originalID)

		case models.TypeRename:
			var req models.RenameRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid rename request format", originalID)
				continue
			}

			s.handleRename(conn, req, originalID)

		case models.TypePing:
			// Handle ping message - simple connection check
			s.handlePing(conn, sigMsg.Payload, originalID)

		case models.TypeGetUserRooms:
			var req models.GetUserRoomsRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid get user rooms request format", originalID)
				continue
			}

			s.handleGetUserRooms(conn, req, originalID)

		default:
			logger.Warn("Unknown message type", "type", sigMsg.Type)
			if originalID != "" {
				s.sendErrorSignal(conn, "Unknown message type", originalID)
			}
		}
	}
}

// sendErrorSignal sends an error message using the SignalingMessage structure
func (s *WebSocketServer) sendErrorSignal(conn *websocket.Conn, errorMsg string, originalID string) {
	errPayload, _ := json.Marshal(map[string]string{"error": errorMsg})

	conn.WriteJSON(models.SignalingMessage{
		ID:      originalID,
		Type:    models.TypeError,
		Payload: errPayload,
	})
}

// sendSignal is a helper function to send a response using SignalingMessage structure
func (s *WebSocketServer) sendSignal(conn *websocket.Conn, msgType models.MessageType, payload interface{}, originalID string) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return conn.WriteJSON(models.SignalingMessage{
		ID:      originalID,
		Type:    msgType,
		Payload: payloadBytes,
	})
}

// handleCreateRoom processes a request to create a new room
func (s *WebSocketServer) handleCreateRoom(conn *websocket.Conn, req models.CreateRoomRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.RoomName == "" || req.Password == "" || req.PublicKey == "" {
		s.sendErrorSignal(conn, "Room name, password, and public key are required", originalID)
		return
	}

	if !s.passwordRegex.MatchString(req.Password) {
		s.sendErrorSignal(conn, "Password does not match required pattern", originalID)
		return
	}

	// Verifica se a chave pública já tem uma sala
	hasRoom, existingRoomID, err := s.supabaseManager.PublicKeyHasRoom(req.PublicKey)
	if err != nil {
		logger.Error("Error checking if public key has a room", "error", err)
	} else if hasRoom {
		s.sendErrorSignal(conn, fmt.Sprintf("This public key has already created room: %s", existingRoomID), originalID)
		return
	}

	roomID := models.GenerateRoomID()

	// Verifica se o ID da sala já existe
	exists, err := s.supabaseManager.RoomExists(roomID)
	if err != nil {
		logger.Error("Error checking if room exists", "error", err)
	} else if exists {
		s.sendErrorSignal(conn, "Room ID conflict, please try again", originalID)
		return
	}

	// Validar e analisar a chave pública Ed25519
	pubKey, err := crypto_utils.ParsePublicKey(req.PublicKey)
	if err != nil {
		s.sendErrorSignal(conn, "Invalid public key format", originalID)
		return
	}

	// Cria uma estrutura ServerRoom apenas para persistência
	room := ServerRoom{
		Room: models.Room{
			ID:       roomID,
			Name:     req.RoomName,
			Password: req.Password,
		},
		PublicKey:    pubKey,
		PublicKeyB64: req.PublicKey,
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
	}

	// Store room in Supabase
	err = s.supabaseManager.CreateRoom(room)
	if err != nil {
		logger.Error("Error creating room in Supabase", "error", err)
		s.sendErrorSignal(conn, "Error creating room in database", originalID)
		return
	}

	// Armazena a chave pública associada a esta conexão
	s.clientToPublicKey[conn] = req.PublicKey

	// Associa este cliente à sala
	s.clients[conn] = roomID

	// Adiciona o cliente à lista de conexões desta sala
	if _, exists := s.networks[roomID]; !exists {
		s.networks[roomID] = []*websocket.Conn{}
	}
	s.networks[roomID] = append(s.networks[roomID], conn)

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		logger.Info("Room created",
			"roomID", roomID,
			"roomName", req.RoomName,
			"clientAddr", conn.RemoteAddr().String())
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	// Prepare response payload
	responsePayload := map[string]interface{}{
		"room_id":    roomID,
		"room_name":  req.RoomName,
		"password":   req.Password,
		"public_key": req.PublicKey,
	}

	s.sendSignal(conn, models.TypeRoomCreated, responsePayload, originalID)
}

// handleJoinRoom processes a request to join an existing room
func (s *WebSocketServer) handleJoinRoom(conn *websocket.Conn, req models.JoinRoomRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fetch room from Supabase - sempre busca mais recente
	room, err := s.supabaseManager.GetRoom(req.RoomID)
	if err != nil {
		s.sendErrorSignal(conn, "Room does not exist", originalID)
		return
	}

	if req.Password != room.Password {
		s.sendErrorSignal(conn, "Incorrect password", originalID)
		return
	}

	if req.PublicKey == "" {
		s.sendErrorSignal(conn, "Public key is required", originalID)
		return
	}

	// Check if room is full - apenas com base nas conexões ativas
	connections := s.networks[req.RoomID]
	if len(connections) >= s.config.MaxClientsPerRoom {
		s.sendErrorSignal(conn, "Room is full", originalID)
		return
	}

	// Check if user is already a member of this room
	isInRoom, err := s.supabaseManager.IsUserInRoom(req.RoomID, req.PublicKey)
	if err != nil {
		logger.Error("Error checking if user is in room", "error", err)
	}

	// If user is not already a member, add them to the user_rooms table
	if !isInRoom {
		err = s.supabaseManager.AddUserToRoom(req.RoomID, req.PublicKey, req.Username)
		if err != nil {
			logger.Error("Error adding user to room", "error", err)
			// Continue anyway, as this is not a critical error
		}
	} else {
		// Update the user's connection status to connected
		err = s.supabaseManager.UpdateUserRoomConnection(req.RoomID, req.PublicKey, true)
		if err != nil {
			logger.Error("Error updating user room connection", "error", err)
			// Continue anyway, as this is not a critical error
		}
	}

	// Armazena a chave pública do cliente que entrou na sala
	s.clientToPublicKey[conn] = req.PublicKey

	// Adiciona o cliente à sala
	s.clients[conn] = req.RoomID
	if _, exists := s.networks[req.RoomID]; !exists {
		s.networks[req.RoomID] = []*websocket.Conn{}
	}
	s.networks[req.RoomID] = append(s.networks[req.RoomID], conn)

	// Atualiza o contador de clientes apenas para o log
	clientCount := len(s.networks[req.RoomID])

	// Update last activity time in Supabase
	err = s.supabaseManager.UpdateRoomActivity(req.RoomID)
	if err != nil && (s.config.LogLevel == "debug") {
		logger.Debug("Error updating room activity", "error", err)
	}

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		logger.Info("Client joined room",
			"clientAddr", conn.RemoteAddr().String(),
			"roomID", req.RoomID,
			"activeClients", clientCount)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	// Envia a resposta ao cliente
	responsePayload := map[string]interface{}{
		"room_id":   req.RoomID,
		"room_name": room.Name,
	}
	s.sendSignal(conn, models.TypeRoomJoined, responsePayload, originalID)

	// Notifica os outros participantes da sala sobre o novo cliente
	for _, peer := range s.networks[req.RoomID] {
		if peer != conn {
			// Informa aos peers que um novo cliente entrou
			peerJoinedPayload := map[string]interface{}{
				"room_id":    req.RoomID,
				"public_key": req.PublicKey,
				"username":   req.Username,
			}
			s.sendSignal(peer, models.TypePeerJoined, peerJoinedPayload, "")

			// Informa ao novo cliente sobre os peers existentes
			peerPublicKey, hasPeerKey := s.clientToPublicKey[peer]
			if hasPeerKey {
				existingPeerPayload := map[string]interface{}{
					"room_id":    req.RoomID,
					"public_key": peerPublicKey,
					"username":   "Peer", // Default username
				}
				s.sendSignal(conn, models.TypePeerJoined, existingPeerPayload, "")
			}
		}
	}
}

// handleConnectRoom processes a request to connect to a previously joined room
// This allows a client to connect to a room without providing the password again
func (s *WebSocketServer) handleConnectRoom(conn *websocket.Conn, req models.ConnectRoomRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fetch room from Supabase
	room, err := s.supabaseManager.GetRoom(req.RoomID)
	if err != nil {
		s.sendErrorSignal(conn, "Room does not exist", originalID)
		return
	}

	if req.PublicKey == "" {
		s.sendErrorSignal(conn, "Public key is required", originalID)
		return
	}

	// Check if user is a member of this room
	isInRoom, err := s.supabaseManager.IsUserInRoom(req.RoomID, req.PublicKey)
	if err != nil {
		logger.Error("Error checking if user is in room", "error", err)
		s.sendErrorSignal(conn, "Error verifying room membership", originalID)
		return
	}

	if !isInRoom {
		s.sendErrorSignal(conn, "You must join this room first", originalID)
		return
	}

	// Check if room is full
	connections := s.networks[req.RoomID]
	if len(connections) >= s.config.MaxClientsPerRoom {
		s.sendErrorSignal(conn, "Room is full", originalID)
		return
	}

	// Update the user's connection status in the database
	err = s.supabaseManager.UpdateUserRoomConnection(req.RoomID, req.PublicKey, true)
	if err != nil {
		logger.Error("Error updating user room connection", "error", err)
		// Continue anyway as this is not critical
	}

	// Store the client's public key
	s.clientToPublicKey[conn] = req.PublicKey

	// Add the client to the room
	s.clients[conn] = req.RoomID
	if _, exists := s.networks[req.RoomID]; !exists {
		s.networks[req.RoomID] = []*websocket.Conn{}
	}
	s.networks[req.RoomID] = append(s.networks[req.RoomID], conn)

	// Update last activity time in Supabase
	err = s.supabaseManager.UpdateRoomActivity(req.RoomID)
	if err != nil && (s.config.LogLevel == "debug") {
		logger.Debug("Error updating room activity", "error", err)
	}

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		logger.Info("Client connected to room (reconnect)",
			"clientAddr", conn.RemoteAddr().String(),
			"roomID", req.RoomID,
			"activeClients", len(s.networks[req.RoomID]))
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	// Send the response to the client
	responsePayload := map[string]interface{}{
		"room_id":   req.RoomID,
		"room_name": room.Name,
	}
	s.sendSignal(conn, models.TypeRoomConnected, responsePayload, originalID)

	// Notify other participants in the room about the new peer connection
	for _, peer := range s.networks[req.RoomID] {
		if peer != conn {
			// Inform peers that a new client connected
			peerConnectedPayload := map[string]interface{}{
				"room_id":    req.RoomID,
				"public_key": req.PublicKey,
				"username":   req.Username,
			}
			s.sendSignal(peer, models.TypePeerConnected, peerConnectedPayload, "")

			// Inform the new client about existing peers
			peerPublicKey, hasPeerKey := s.clientToPublicKey[peer]
			if hasPeerKey {
				existingPeerPayload := map[string]interface{}{
					"room_id":    req.RoomID,
					"public_key": peerPublicKey,
					"username":   "Peer", // Default username
				}
				s.sendSignal(conn, models.TypePeerConnected, existingPeerPayload, "")
			}
		}
	}
}

// handleDisconnectRoom processes a request to disconnect from a room without leaving it
// This allows clients to disconnect temporarily but remain as members of the room
func (s *WebSocketServer) handleDisconnectRoom(conn *websocket.Conn, req models.DisconnectRoomRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID := req.RoomID

	// If roomID wasn't provided, use the roomID associated with this connection
	if roomID == "" {
		roomID = s.clients[conn]
		if roomID == "" {
			// Client isn't in any room
			s.sendErrorSignal(conn, "Not connected to any room", originalID)
			return
		}
	}

	// Verify the client is actually in this room
	if s.clients[conn] != roomID {
		s.sendErrorSignal(conn, "Not connected to this room", originalID)
		return
	}

	// Get the public key before removing the client
	publicKey, hasPublicKey := s.clientToPublicKey[conn]
	if !hasPublicKey {
		s.sendErrorSignal(conn, "Public key not found for this connection", originalID)
		return
	}

	// Check if this client is the room owner
	room, err := s.supabaseManager.GetRoom(roomID)
	if err != nil {
		s.sendErrorSignal(conn, "Room not found", originalID)
		return
	}

	isOwner := (publicKey == room.PublicKeyB64)

	// If this is the owner, we need to update the room activity timestamp
	// to ensure the room doesn't get deleted during cleanup
	if isOwner {
		err = s.supabaseManager.UpdateRoomActivity(roomID)
		if err != nil && (s.config.LogLevel == "debug") {
			logger.Debug("Error updating room activity", "error", err)
		}
	}

	// Update the user's connection status in the database
	err = s.supabaseManager.UpdateUserRoomConnection(roomID, publicKey, false)
	if err != nil {
		logger.Error("Error updating user room connection status", "error", err)
		// Continue anyway as this is not critical
	}

	// Remove client from the networks map but DO NOT remove from the clients map
	// This allows the server to remember which room the client belongs to
	if networks, exists := s.networks[roomID]; exists {
		for i, peer := range networks {
			if peer == conn {
				s.networks[roomID] = append(networks[:i], networks[i+1:]...)
				break
			}
		}

		// If no more clients in the room's network list, remove the network entry
		// but don't delete the room from Supabase
		if len(s.networks[roomID]) == 0 {
			delete(s.networks, roomID)
		} else {
			// Notify other participants that this client disconnected
			for _, peer := range s.networks[roomID] {
				peerDisconnectedPayload := map[string]interface{}{
					"room_id":    roomID,
					"public_key": publicKey,
				}
				s.sendSignal(peer, models.TypePeerDisconnected, peerDisconnectedPayload, "")
			}
		}
	}

	// Send disconnect confirmation
	disconnectResponse := map[string]interface{}{
		"room_id": roomID,
	}
	s.sendSignal(conn, models.TypeRoomDisconnected, disconnectResponse, originalID)

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		logger.Info("Client disconnected from room (but still a member)",
			"clientAddr", conn.RemoteAddr().String(),
			"roomID", roomID,
			"isOwner", isOwner)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))
}

// handleLeaveRoom processes a request from a client to leave a room
func (s *WebSocketServer) handleLeaveRoom(conn *websocket.Conn, req models.LeaveRoomRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID := req.RoomID

	// Se o roomID não foi fornecido, usa o roomID associado a esta conexão
	if roomID == "" {
		roomID = s.clients[conn]
		if roomID == "" {
			// Cliente não está em nenhuma sala
			s.sendErrorSignal(conn, "Not connected to any room", originalID)
			return
		}
	}

	// Recupera a chave pública do cliente
	publicKey := req.PublicKey
	if publicKey == "" {
		publicKey, _ = s.clientToPublicKey[conn]
		if publicKey == "" {
			s.sendErrorSignal(conn, "Public key is required", originalID)
			return
		}
	}

	// Busca a sala para verificar se o cliente é o dono
	room, err := s.supabaseManager.GetRoom(roomID)
	if err != nil {
		s.sendErrorSignal(conn, "Room not found", originalID)
		return
	}

	// Verifica se o cliente é o dono da sala
	isCreator := (publicKey == room.PublicKeyB64)

	// Remove the user from the user_rooms table
	err = s.supabaseManager.RemoveUserFromRoom(roomID, publicKey)
	if err != nil {
		logger.Error("Error removing user from user_rooms table", "error", err)
		// Continue anyway, as this is not a critical error
	}

	// Se for o dono da sala e não for para preservar
	if isCreator {
		logger.Info("Room owner leaving", "roomID", roomID, "intentionalDelete", true)

		// Notificar todos os outros clientes que a sala foi excluída
		for _, peer := range s.networks[roomID] {
			if peer != conn {
				// Usando o struct correto do models para TypeRoomDeleted
				deletedNotification := models.RoomDeletedNotification{
					RoomID: roomID,
				}
				s.sendSignal(peer, models.TypeRoomDeleted, deletedNotification, "")
			}
		}

		// Excluir a sala do Supabase
		err := s.supabaseManager.DeleteRoom(roomID)
		if err != nil {
			logger.Error("Error deleting room from Supabase", "error", err)
		}

		// Limpar todas as referências da sala em memória
		delete(s.networks, roomID)
		for c, cRoomID := range s.clients {
			if cRoomID == roomID {
				delete(s.clients, c)
			}
		}

		logger.Info("Room deleted because owner left", "roomID", roomID)
	} else {
		// Se não for o dono, apenas remove o cliente da sala
		s.removeClient(conn, roomID)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	// Confirma a saída
	leaveSuccessPayload := map[string]interface{}{
		"room_id": roomID,
	}
	s.sendSignal(conn, models.TypeLeaveRoom, leaveSuccessPayload, originalID)

	logger.Info("Client left room via explicit leave",
		"clientAddr", conn.RemoteAddr().String(),
		"roomID", roomID,
		"isCreator", isCreator)
}

// handleKick processes a request to kick a user from the room
func (s *WebSocketServer) handleKick(conn *websocket.Conn, req models.KickRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.supabaseManager.GetRoom(req.RoomID)
	if err != nil {
		s.sendErrorSignal(conn, "Room does not exist", originalID)
		return
	}

	// Verifica se o cliente é o dono da sala
	publicKey, hasPublicKey := s.clientToPublicKey[conn]
	if !hasPublicKey || publicKey != room.PublicKeyB64 {
		s.sendErrorSignal(conn, "Only room owner can kick users", originalID)
		return
	}

	for _, peer := range s.networks[req.RoomID] {
		if peer.RemoteAddr().String() == req.TargetID {
			kickedPayload := map[string]interface{}{
				"room_id": req.RoomID,
			}
			s.sendSignal(peer, models.TypeKicked, kickedPayload, "")

			peer.Close()
			s.removeClient(peer, req.RoomID)
			logger.Info("Client kicked from room", "targetID", req.TargetID, "roomID", req.RoomID)

			s.statsManager.UpdateStats(len(s.clients), len(s.networks))

			kickSuccessPayload := map[string]interface{}{
				"room_id":   req.RoomID,
				"target_id": req.TargetID,
			}
			s.sendSignal(conn, models.TypeKickSuccess, kickSuccessPayload, originalID)
			return
		}
	}

	s.sendErrorSignal(conn, "Target client not found", originalID)
}

// handleRename processes a request to rename a room
func (s *WebSocketServer) handleRename(conn *websocket.Conn, req models.RenameRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.supabaseManager.GetRoom(req.RoomID)
	if err != nil {
		s.sendErrorSignal(conn, "Room does not exist", originalID)
		return
	}

	// Verifica se o cliente é o dono da sala
	publicKey, hasPublicKey := s.clientToPublicKey[conn]
	if !hasPublicKey || publicKey != room.PublicKeyB64 {
		s.sendErrorSignal(conn, "Only room owner can rename the room", originalID)
		return
	}

	err = s.supabaseManager.UpdateRoomName(req.RoomID, req.RoomName)
	if err != nil {
		logger.Error("Error updating room name", "error", err)
		s.sendErrorSignal(conn, "Error updating room name in database", originalID)
		return
	}

	logger.Info("Room renamed", "roomID", req.RoomID, "newName", req.RoomName)

	// Notify all clients in the room about the rename
	renamePayload := map[string]interface{}{
		"room_id":   req.RoomID,
		"room_name": req.RoomName,
	}

	for _, peer := range s.networks[req.RoomID] {
		s.sendSignal(peer, models.TypeRoomRenamed, renamePayload, "")
	}

	// Additional successful rename notification to the requester
	s.sendSignal(conn, models.TypeRenameSuccess, renamePayload, originalID)
}

// validatePublicKey processa a chave pública fornecida na mensagem
func (s *WebSocketServer) validatePublicKey(publicKeyBase64 string) (bool, ed25519.PublicKey, error) {
	if publicKeyBase64 == "" {
		logger.Warn("Empty public key received")
		return false, nil, errors.New("chave pública vazia")
	}

	logger.Debug("Validating public key",
		"keyPrefix", publicKeyBase64[:10]+"...",
		"keyLength", len(publicKeyBase64))

	pubKey, err := crypto_utils.ParsePublicKey(publicKeyBase64)
	if err != nil {
		logger.Error("Failed to parse public key", "error", err)
		return false, nil, fmt.Errorf("falha ao processar chave pública: %w", err)
	}

	logger.Debug("Public key validated successfully", "keySize", len(pubKey))
	return true, pubKey, nil
}

// handleDisconnect manages cleanup when a client disconnects
// Logic: Remove client from room tracking, update Supabase for room owners, clean up resources
func (s *WebSocketServer) handleDisconnect(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Quando ocorre uma desconexão (fechamento do app ou perda de conexão),
	roomID := s.clients[conn]
	if roomID != "" {
		// Recupera a chave pública do cliente antes de removê-lo
		publicKey, hasPublicKey := s.clientToPublicKey[conn]
		isOwner := false

		// Verifica se o cliente que está saindo é o dono da sala com base na chave pública
		if hasPublicKey {
			// Busca a sala no banco de dados
			room, err := s.supabaseManager.GetRoom(roomID)
			if err == nil && publicKey == room.PublicKeyB64 {
				// A chave pública do cliente que está saindo coincide com a do criador da sala
				isOwner = true
				logger.Info("Room owner disconnected",
					"keyPrefix", publicKey[:10]+"...")

				// Update room activity to prevent it from being removed during cleanup
				err := s.supabaseManager.UpdateRoomActivity(roomID)
				if err != nil {
					logger.Error("Error updating room activity", "error", err)
				}
			}
		}

		// Notifica outros membros da sala sobre a saída deste cliente
		for _, peer := range s.networks[roomID] {
			if peer != conn {
				peerLeftPayload := map[string]interface{}{
					"room_id":    roomID,
					"public_key": publicKey,
				}
				s.sendSignal(peer, models.TypePeerLeft, peerLeftPayload, "")
			}
		}

		// Limpa as referências do cliente
		delete(s.clients, conn)
		delete(s.clientToPublicKey, conn)

		// Se a sala não existe no networks, não há nada mais a fazer
		network, exists := s.networks[roomID]
		if !exists {
			return
		}

		// Remove o cliente da lista de conexões da sala
		for i, peer := range network {
			if peer == conn {
				s.networks[roomID] = append(network[:i], network[i+1:]...)
				break
			}
		}

		// Se não houver mais clientes na sala, remove apenas da memória mas mantém no banco
		if len(s.networks[roomID]) == 0 {
			delete(s.networks, roomID)
		}

		if isOwner {
			logger.Info("Owner disconnected but room preserved",
				"clientAddr", conn.RemoteAddr().String(),
				"roomID", roomID)
		} else {
			logger.Info("Client disconnected from room",
				"clientAddr", conn.RemoteAddr().String(),
				"roomID", roomID)
		}
	} else {
		// Cliente não estava em nenhuma sala
		delete(s.clientToPublicKey, conn)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))
}

// DeleteStaleRooms removes rooms that have not been active for a specified period
// Logic: Query for rooms that haven't been active past the expiry period and delete them
func (s *WebSocketServer) DeleteStaleRooms() {
	staleRooms, err := s.supabaseManager.GetStaleRooms(s.config.RoomExpiryDays)
	if err != nil {
		logger.Error("Error fetching stale rooms", "error", err)
		return
	}

	numRemoved := 0
	for _, room := range staleRooms {
		err := s.supabaseManager.DeleteRoom(room.ID)
		if err != nil {
			logger.Error("Error deleting stale room", "roomID", room.ID, "error", err)
		} else {
			logger.Info("Deleted stale room", "roomID", room.ID)
			numRemoved++
		}
	}

	s.statsManager.UpdateCleanupStats(numRemoved)
}

// Start initializes and starts the WebSocket server
// Logic: Set up HTTP handlers, start room cleanup routine, and listen for incoming connections
func (s *WebSocketServer) Start(port string) error {
	mux := http.NewServeMux()

	// Add handlers to the mux
	mux.HandleFunc("/ws", s.HandleWebSocketEndpoint)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Add stats endpoint
	mux.HandleFunc("/stats", s.handleStatsEndpoint)

	// Create an HTTP server with the mux
	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Periodically delete stale rooms
	go func() {
		ticker := time.NewTicker(s.config.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.DeleteStaleRooms()
			case <-s.shutdownChan:
				return // Stop cleanup goroutine when server shuts down
			}
		}
	}()

	logger.Info("WebSocket server starting", "port", port)

	// Start HTTP server in a separate goroutine so we can return errors
	go func() {
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// handlePing processes ping messages from clients and responds with a pong
// This allows clients to verify their connection to the server
func (s *WebSocketServer) handlePing(conn *websocket.Conn, payload []byte, originalID string) {
	// Parse the ping message
	var pingData map[string]interface{}
	if err := json.Unmarshal(payload, &pingData); err != nil {
		logger.Error("Error parsing ping payload", "error", err)
		s.sendErrorSignal(conn, "Invalid ping format", originalID)
		return
	}

	// Create response payload with server timestamp
	pongPayload := map[string]interface{}{
		"client_timestamp": pingData["timestamp"],
		"server_timestamp": time.Now().UnixNano(),
		"status":           "ok",
	}

	// Log ping if in debug mode
	if s.config.LogLevel == "debug" {
		clientAddr := conn.RemoteAddr().String()
		logger.Debug("Received ping from client", "clientAddr", clientAddr)
	}

	// Send pong response with the same message ID
	s.sendSignal(conn, models.TypePing, pongPayload, originalID)
}

// removeClient remove um cliente da sala e, se necessário, a sala do Supabase
// Logic: Clean up client references and potentially delete room if owner leaves
func (s *WebSocketServer) removeClient(conn *websocket.Conn, roomID string) {
	// Se o roomID não foi fornecido, não há nada a fazer
	if roomID == "" {
		// Limpa a chave pública do cliente
		delete(s.clientToPublicKey, conn)
		return
	}

	// Recupera a chave pública do cliente antes de removê-lo
	publicKey, hasPublicKey := s.clientToPublicKey[conn]

	// Limpa as referências do cliente
	delete(s.clients, conn)
	delete(s.clientToPublicKey, conn)

	// Se a sala não existe no networks, não há nada mais a fazer
	network, exists := s.networks[roomID]
	if !exists {
		return
	}

	// Remove o cliente da lista de conexões da sala
	for i, peer := range network {
		if peer == conn {
			s.networks[roomID] = append(network[:i], network[i+1:]...)
			break
		}
	}

	// Verifica se o cliente que está saindo é o dono da sala com base na chave pública
	isCreator := false
	if hasPublicKey {
		// Busca a sala no banco de dados
		room, err := s.supabaseManager.GetRoom(roomID)
		if err == nil && publicKey == room.PublicKeyB64 {
			// A chave pública do cliente que está saindo coincide com a do criador da sala
			isCreator = true
			logger.Info("Room creator disconnected", "publicKey", publicKey)
		}
	}

	// Se for o criador, deleta a sala do banco de dados
	if isCreator {
		err := s.supabaseManager.DeleteRoom(roomID)
		if err != nil && (s.config.LogLevel == "debug") {
			logger.Debug("Error deleting room from Supabase on creator disconnect", "error", err)
		}

		// Notifica todos os outros participantes da sala que ela foi excluída
		for _, peer := range s.networks[roomID] {
			if peer != conn {
				// Usando o struct correto do models para TypeRoomDeleted
				deletedNotification := models.RoomDeletedNotification{
					RoomID: roomID,
				}
				s.sendSignal(peer, models.TypeRoomDeleted, deletedNotification, "")
			}
		}

		// Remove completamente a sala e seus clientes
		delete(s.networks, roomID)
		for c, cRoomID := range s.clients {
			if cRoomID == roomID {
				delete(s.clients, c)
			}
		}
		logger.Info("Room deleted because owner disconnected", "roomID", roomID)
	} else if len(s.networks[roomID]) == 0 {
		// Se não houver mais clientes na sala, remove a referência da sala
		delete(s.networks, roomID)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	logger.Info("Client left room", "clientAddr", conn.RemoteAddr().String(), "roomID", roomID)
}

// handleStatsEndpoint is the HTTP handler for the /stats endpoint
func (s *WebSocketServer) handleStatsEndpoint(w http.ResponseWriter, r *http.Request) {
	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	w.Header().Set("Content-Type", "application/json")

	// Create response with current statistics
	statsResponse := map[string]interface{}{
		"server_stats": s.statsManager.GetStats(),
		"config": map[string]interface{}{
			"max_clients_per_room": s.config.MaxClientsPerRoom,
			"room_expiry_days":     s.config.RoomExpiryDays,
			"cleanup_interval":     s.config.CleanupInterval.String(),
			"allow_all_origins":    s.config.AllowAllOrigins,
		},
	}

	// Convert to JSON and send response
	jsonBytes, err := json.Marshal(statsResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to generate statistics"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

// handleGetUserRooms processes a request to get all rooms a user has joined
func (s *WebSocketServer) handleGetUserRooms(conn *websocket.Conn, req models.GetUserRoomsRequest, originalID string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if req.PublicKey == "" {
		s.sendErrorSignal(conn, "Public key is required", originalID)
		return
	}

	// Get all rooms for this user
	userRooms, err := s.supabaseManager.GetUserRooms(req.PublicKey)
	if err != nil {
		logger.Error("Error fetching user rooms", "error", err, "publicKey", req.PublicKey)
		s.sendErrorSignal(conn, "Error fetching user rooms", originalID)
		return
	}

	// Build response with room details
	response := models.UserRoomsResponse{
		Rooms: make([]models.UserRoomInfo, 0, len(userRooms)),
	}

	for _, userRoom := range userRooms {
		// Get room details
		room, err := s.supabaseManager.GetRoom(userRoom.RoomID)
		if err != nil {
			// Skip rooms that no longer exist
			logger.Debug("Room no longer exists", "roomID", userRoom.RoomID)
			continue
		}

		roomInfo := models.UserRoomInfo{
			RoomID:        userRoom.RoomID,
			RoomName:      room.Name,
			IsConnected:   userRoom.IsConnected,
			JoinedAt:      userRoom.JoinedAt,
			LastConnected: userRoom.LastConnected,
		}
		response.Rooms = append(response.Rooms, roomInfo)
	}

	if s.config.LogLevel == "debug" {
		logger.Debug("Sending user rooms",
			"publicKey", req.PublicKey,
			"roomCount", len(response.Rooms))
	}

	// Send response
	s.sendSignal(conn, models.TypeUserRooms, response, originalID)
}

// InitiateGracefulShutdown starts the graceful shutdown process
func (s *WebSocketServer) InitiateGracefulShutdown(timeout time.Duration, restartInfo string) {
	if s.isShutdown {
		logger.Warn("Shutdown already in progress")
		return
	}

	s.isShutdown = true
	logger.Info("Initiating graceful shutdown", "timeout", timeout)

	// Wait a short amount of time for in-flight requests to complete
	time.Sleep(500 * time.Millisecond)

	// First step: Notify all connected clients about impending shutdown
	s.notifyClientsAboutShutdown(int(timeout.Seconds()), restartInfo)

	// Allow some time for notification messages to be sent
	time.Sleep(1 * time.Second)

	// Set a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Start persisting server state
	s.persistStateForRestart()

	// Actually shutdown the HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			logger.Error("HTTP server shutdown error", "error", err)
		}
	}

	// Signal successful shutdown
	close(s.shutdownChan)
	logger.Info("Graceful shutdown completed")
}

// notifyClientsAboutShutdown sends a shutdown notification to all connected clients
func (s *WebSocketServer) notifyClientsAboutShutdown(shutdownSeconds int, restartInfo string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logger.Info("Notifying clients about server shutdown", "clientCount", len(s.clients))

	notification := models.ServerShutdownNotification{
		Message:     "Server is shutting down for maintenance",
		ShutdownIn:  shutdownSeconds,
		RestartInfo: restartInfo,
	}

	// Send notification to all clients
	for conn := range s.clients {
		// Generate a random message ID
		msgID, err := models.GenerateMessageID()
		if err != nil {
			msgID = ""
		}

		err = s.sendSignal(conn, models.TypeServerShutdown, notification, msgID)
		if err != nil {
			logger.Error("Error notifying client", "clientAddr", conn.RemoteAddr().String(), "error", err)
		}
	}
}

// persistStateForRestart saves the current server state to enable a clean restart
func (s *WebSocketServer) persistStateForRestart() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logger.Info("Persisting state for potential server restart")

	// Update all active room timestamps in the database
	for roomID := range s.networks {
		// Ensure this room's activity is updated to prevent cleanup
		err := s.supabaseManager.UpdateRoomActivity(roomID)
		if err != nil {
			logger.Error("Error updating room activity", "roomID", roomID, "error", err)
		} else {
			logger.Info("Room state persisted", "roomID", roomID)
		}
	}

	// Could add more state persistence here if needed
	logger.Info("Server state persistence completed")
}

// WaitForShutdown blocks until the server has shut down
func (s *WebSocketServer) WaitForShutdown() {
	<-s.shutdownChan
}
