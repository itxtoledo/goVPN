// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/server/websocket_server.go
package main

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

	return &WebSocketServer{
		clients:           make(map[*websocket.Conn]string),
		networks:          make(map[string][]*websocket.Conn),
		clientToPublicKey: make(map[*websocket.Conn]string),
		config:            cfg,
		supabaseManager:   supaMgr,
		upgrader:          upgrader,
		passwordRegex:     passwordRegex,
	}, nil
}

// handleWebSocketEndpoint is the HTTP handler for WebSocket connections
// Logic: Upgrade HTTP connection to WebSocket, then handle incoming messages in a loop
// until the connection is closed
func (s *WebSocketServer) HandleWebSocketEndpoint(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	for {
		var sigMsg models.SignalingMessage
		err := conn.ReadJSON(&sigMsg)
		if err != nil {
			s.handleDisconnect(conn)
			return
		}

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

		default:
			log.Printf("Unknown message type: %s", sigMsg.Type)
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
		log.Printf("Error checking if public key has a room: %v", err)
	} else if hasRoom {
		s.sendErrorSignal(conn, fmt.Sprintf("This public key has already created room: %s", existingRoomID), originalID)
		return
	}

	roomID := models.GenerateRoomID()

	// Verifica se o ID da sala já existe
	exists, err := s.supabaseManager.RoomExists(roomID)
	if err != nil {
		log.Printf("Error checking if room exists: %v", err)
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
		log.Printf("Error creating room in Supabase: %v", err)
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
		log.Printf("Room %s (%s) created by %s", roomID, req.RoomName, conn.RemoteAddr().String())
	}

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
		log.Printf("Error updating room activity: %v", err)
	}

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		log.Printf("Client %s joined room %s (active clients: %d)", conn.RemoteAddr().String(), req.RoomID, clientCount)
	}

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
	publicKey, hasPublicKey := s.clientToPublicKey[conn]
	if !hasPublicKey {
		s.sendErrorSignal(conn, "Public key not found for this connection", originalID)
		return
	}

	// Busca a sala para verificar se o cliente é o dono
	room, err := s.supabaseManager.GetRoom(roomID)
	if err != nil {
		s.sendErrorSignal(conn, "Room not found", originalID)
		return
	}

	// Verifica se o cliente é o dono da sala
	isCreator := (publicKey == room.PublicKeyB64)

	// Use the PreserveRoom flag from the message
	preserveRoom := req.PreserveRoom

	// Se for o dono da sala e não for para preservar
	if isCreator && !preserveRoom {
		log.Printf("Room owner is leaving room %s, intentionally deleting the room", roomID)

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
			log.Printf("Error deleting room from Supabase: %v", err)
		}

		// Limpar todas as referências da sala em memória
		delete(s.networks, roomID)
		for c, cRoomID := range s.clients {
			if cRoomID == roomID {
				delete(s.clients, c)
			}
		}

		log.Printf("Room %s deleted because owner left", roomID)
	} else {
		// Se não for o dono, ou for para preservar, apenas remove o cliente da sala
		s.removeClient(conn, roomID, preserveRoom)
	}

	// Confirma a saída
	leaveSuccessPayload := map[string]interface{}{
		"room_id": roomID,
	}
	s.sendSignal(conn, models.TypeLeaveRoom, leaveSuccessPayload, originalID)

	log.Printf("Client %s left room %s via explicit leave (isCreator=%v, preserveRoom=%v)",
		conn.RemoteAddr().String(), roomID, isCreator, preserveRoom)
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
			s.removeClient(peer, req.RoomID, false)
			log.Printf("Client %s kicked from room %s", req.TargetID, req.RoomID)

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
		log.Printf("Error updating room name: %v", err)
		s.sendErrorSignal(conn, "Error updating room name in database", originalID)
		return
	}

	log.Printf("Room %s renamed to %s", req.RoomID, req.RoomName)

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
		log.Printf("Empty public key received")
		return false, nil, errors.New("chave pública vazia")
	}

	log.Printf("Validating public key: %s (length: %d)", publicKeyBase64[:10]+"...", len(publicKeyBase64))

	pubKey, err := crypto_utils.ParsePublicKey(publicKeyBase64)
	if err != nil {
		log.Printf("Failed to parse public key: %v", err)
		return false, nil, fmt.Errorf("falha ao processar chave pública: %w", err)
	}

	log.Printf("Public key validated successfully, size: %d bytes", len(pubKey))
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
				log.Printf("Room owner with public key %s disconnected", publicKey[:10]+"...")

				// Update room activity to prevent it from being removed during cleanup
				err := s.supabaseManager.UpdateRoomActivity(roomID)
				if err != nil {
					log.Printf("Error updating room activity: %v", err)
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
			log.Printf("Owner %s disconnected but room %s preserved", conn.RemoteAddr().String(), roomID)
		} else {
			log.Printf("Client %s disconnected from room %s", conn.RemoteAddr().String(), roomID)
		}
	} else {
		// Cliente não estava em nenhuma sala
		delete(s.clientToPublicKey, conn)
	}
}

// DeleteStaleRooms removes rooms that have not been active for a specified period
// Logic: Query for rooms that haven't been active past the expiry period and delete them
func (s *WebSocketServer) DeleteStaleRooms() {
	staleRooms, err := s.supabaseManager.GetStaleRooms(s.config.RoomExpiryDays)
	if err != nil {
		log.Printf("Error fetching stale rooms: %v", err)
		return
	}

	for _, room := range staleRooms {
		err := s.supabaseManager.DeleteRoom(room.ID)
		if err != nil {
			log.Printf("Error deleting stale room %s: %v", room.ID, err)
		} else {
			log.Printf("Deleted stale room %s", room.ID)
		}
	}
}

// Start initializes and starts the WebSocket server
// Logic: Set up HTTP handlers, start room cleanup routine, and listen for incoming connections
func (s *WebSocketServer) Start(port string) error {
	http.HandleFunc("/ws", s.HandleWebSocketEndpoint)

	// Add health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Periodically delete stale rooms
	go func() {
		ticker := time.NewTicker(s.config.CleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			s.DeleteStaleRooms()
		}
	}()

	log.Printf("WebSocket server starting on :%s", port)
	return http.ListenAndServe(":"+port, nil)
}

// handlePing processes ping messages from clients and responds with a pong
// This allows clients to verify their connection to the server
func (s *WebSocketServer) handlePing(conn *websocket.Conn, payload []byte, originalID string) {
	// Parse the ping message
	var pingData map[string]interface{}
	if err := json.Unmarshal(payload, &pingData); err != nil {
		log.Printf("Error parsing ping payload: %v", err)
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
		log.Printf("Received ping from client %s", clientAddr)
	}

	// Send pong response with the same message ID
	s.sendSignal(conn, models.TypePing, pongPayload, originalID)
}

// removeClient remove um cliente da sala e, se necessário, a sala do Supabase
// Logic: Clean up client references and potentially delete room if owner leaves with preserveRoom=false
func (s *WebSocketServer) removeClient(conn *websocket.Conn, roomID string, preserveRoom bool) {
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
			log.Printf("Room creator disconnected: %s", publicKey)
		}
	}

	// Se for o criador e NÃO for para preservar a sala, deleta a sala do banco de dados
	if isCreator && !preserveRoom {
		err := s.supabaseManager.DeleteRoom(roomID)
		if err != nil && (s.config.LogLevel == "debug") {
			log.Printf("Error deleting room from Supabase on creator disconnect: %v", err)
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
		log.Printf("Room %s deleted because owner disconnected (and preserveRoom=false)", roomID)
	} else if isCreator {
		// Se é o criador mas deve preservar a sala
		log.Printf("Room %s preserved even though owner disconnected", roomID)

		// Atualiza o último acesso para evitar que seja removida pela limpeza de salas antigas
		err := s.supabaseManager.UpdateRoomActivity(roomID)
		if err != nil && (s.config.LogLevel == "debug") {
			log.Printf("Error updating room activity on creator disconnect: %v", err)
		}

		// Se não tiver mais clientes na sala, remove da memória mas mantém no Supabase
		if len(s.networks[roomID]) == 0 {
			delete(s.networks, roomID)
		}
	} else if len(s.networks[roomID]) == 0 {
		// Se não houver mais clientes na sala, remove a referência da sala
		delete(s.networks, roomID)
	}

	log.Printf("Client %s left room %s", conn.RemoteAddr().String(), roomID)
}
