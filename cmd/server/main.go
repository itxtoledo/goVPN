package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/itxtoledo/govpn/libs/models" // Import the models package
	"github.com/joho/godotenv"               // Import godotenv package
	"github.com/supabase-community/supabase-go"
)

// loadEnvFile attempts to load environment variables from a .env file if it exists
func loadEnvFile() {
	// Try to load from .env file
	err := godotenv.Load()
	if err != nil {
		// Only log at debug level since the .env file might not exist, which is normal
		log.Printf("No .env file found or error loading .env file: %v", err)
	} else {
		log.Println("Loaded environment variables from .env file")
	}
}

// Config structure to hold our server configuration
type Config struct {
	Port               string
	AllowAllOrigins    bool
	MaxRooms           int
	PasswordPattern    string
	MaxClientsPerRoom  int
	LogLevel           string
	IdleTimeout        time.Duration
	PingInterval       time.Duration
	ReadBufferSize     int
	WriteBufferSize    int
	SupabaseURL        string
	SupabaseKey        string
	SupabaseRoomsTable string
	CleanupInterval    time.Duration
	RoomExpiryDays     int
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	// First try to load environment variables from a .env file
	loadEnvFile()

	config := Config{
		Port:               getEnv("PORT", "8080"),
		AllowAllOrigins:    getEnvBool("ALLOW_ALL_ORIGINS", true),
		PasswordPattern:    `^\d{4}$`,
		MaxRooms:           getEnvInt("MAX_ROOMS", 100),
		MaxClientsPerRoom:  getEnvInt("MAX_CLIENTS_PER_ROOM", 10),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		IdleTimeout:        time.Second * time.Duration(getEnvInt("IDLE_TIMEOUT_SECONDS", 60)),
		PingInterval:       time.Second * time.Duration(getEnvInt("PING_INTERVAL_SECONDS", 30)),
		ReadBufferSize:     getEnvInt("READ_BUFFER_SIZE", 1024),
		WriteBufferSize:    getEnvInt("WRITE_BUFFER_SIZE", 1024),
		SupabaseURL:        getEnv("SUPABASE_URL", ""),
		SupabaseKey:        getEnv("SUPABASE_KEY", ""),
		SupabaseRoomsTable: getEnv("SUPABASE_ROOMS_TABLE", "rooms"),
		CleanupInterval:    time.Hour * time.Duration(getEnvInt("CLEANUP_INTERVAL_HOURS", 24)),
		RoomExpiryDays:     getEnvInt("ROOM_EXPIRY_DAYS", 30),
	}

	log.Printf("Server configuration loaded: %+v", config)
	return config
}

// Helper functions for environment variables
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	intVal, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Warning: Could not parse %s as integer: %v. Using default: %d", key, err, fallback)
		return fallback
	}
	return intVal
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	value = strings.ToLower(value)
	return value == "true" || value == "1" || value == "yes" || value == "y"
}

var serverConfig Config
var upgrader websocket.Upgrader
var passwordRegex *regexp.Regexp

// ServerRoom extends the basic Room model with server-specific fields
type ServerRoom struct {
	models.Room                 // Embed the Room from models package
	PublicKey    *rsa.PublicKey `json:"-"`          // Not stored in Supabase directly
	PublicKeyB64 string         `json:"public_key"` // Stored as base64 string in Supabase
	CreatedAt    time.Time      `json:"created_at"`
	LastActive   time.Time      `json:"last_active"`
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

// Server manages clients and rooms
type Server struct {
	clients      map[*websocket.Conn]string
	networks     map[string][]*websocket.Conn
	roomCreators map[string]*websocket.Conn
	publicKeys   map[string]string // Maps public key to roomID
	mu           sync.RWMutex
	config       Config
	supabase     *supabase.Client
}

func NewServer(cfg Config) (*Server, error) {
	if cfg.SupabaseURL == "" || cfg.SupabaseKey == "" {
		return nil, errors.New("supabase URL and API key are required")
	}

	supaClient, err := supabase.NewClient(cfg.SupabaseURL, cfg.SupabaseKey, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to create Supabase client: %w", err)
	}

	return &Server{
		clients:      make(map[*websocket.Conn]string),
		networks:     make(map[string][]*websocket.Conn),
		roomCreators: make(map[string]*websocket.Conn),
		publicKeys:   make(map[string]string),
		config:       cfg,
		supabase:     supaClient,
	}, nil
}

// generateRoomID creates a SHA-256 hash based on room name, salt, and timestamp
func generateRoomID(roomName string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	input := fmt.Sprintf("%s:%s:%d", roomName, base64.StdEncoding.EncodeToString(salt), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// GenerateRandomID gera um ID aleatório em formato hexadecimal com o comprimento desejado
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

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	for {
		var msg models.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			s.handleDisconnect(conn)
			return
		}

		// Salva o ID da mensagem original para incluir na resposta
		originalID := msg.MessageID

		// Processa a mensagem com base no tipo
		switch msg.Type {
		case "CreateRoom":
			s.handleCreateRoom(conn, msg)
		case "JoinRoom":
			s.handleJoinRoom(conn, msg)
		case "Offer":
			s.handleOffer(conn, msg)
		case "Answer":
			s.handleAnswer(conn, msg)
		case "Candidate":
			s.handleCandidate(conn, msg)
		case "Kick":
			s.handleKick(conn, msg)
		case "Rename":
			s.handleRename(conn, msg)
		case "Delete":
			s.handleDelete(conn, msg)
		// case "LeaveRoom":
		// 	s.handleLeaveRoom(conn, msg)
		default:
			log.Printf("Unknown message type: %s", msg.Type)
			// Se a mensagem tinha um ID, responde com erro mantendo o mesmo ID
			if originalID != "" {
				responseMsg := models.Message{
					Type:      "Error",
					Data:      []byte(`"Unknown message type"`),
					MessageID: originalID,
				}
				conn.WriteJSON(responseMsg)
			}
		}
	}
}

// Fetch a room from Supabase
func (s *Server) fetchRoom(roomID string) (*ServerRoom, error) {
	var results []SupabaseRoom
	data, _, err := s.supabase.From(s.config.SupabaseRoomsTable).Select("*", "", false).Eq("id", roomID).Execute()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal room data: %w", err)
	}

	if len(results) == 0 {
		return nil, errors.New("room not found")
	}

	// Parse the public key from the base64 string
	pubKeyBytes, err := base64.StdEncoding.DecodeString(results[0].PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key format: %v", err)
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}

	room := &ServerRoom{
		Room: models.Room{
			ID:       results[0].ID,
			Name:     results[0].Name,
			Password: results[0].Password,
		},
		PublicKey:    rsaPubKey,
		PublicKeyB64: results[0].PublicKey,
		CreatedAt:    results[0].CreatedAt,
		LastActive:   results[0].LastActive,
	}

	return room, nil
}

// Create a room in Supabase
func (s *Server) createRoomInSupabase(room ServerRoom) error {
	supaRoom := SupabaseRoom{
		ID:         room.ID,
		Name:       room.Name,
		Password:   room.Password,
		PublicKey:  room.PublicKeyB64,
		CreatedAt:  room.CreatedAt,
		LastActive: room.LastActive,
	}

	_, _, err := s.supabase.From(s.config.SupabaseRoomsTable).Insert(supaRoom, false, "", "", "").Execute()
	return err
}

// Delete a room from Supabase
func (s *Server) deleteRoomFromSupabase(roomID string) error {
	_, _, err := s.supabase.From(s.config.SupabaseRoomsTable).Delete("", "").Eq("id", roomID).Execute()
	return err
}

// Update room last activity time in Supabase
func (s *Server) updateRoomActivity(roomID string) error {
	updates := map[string]interface{}{
		"last_active": time.Now(),
	}
	_, _, err := s.supabase.From(s.config.SupabaseRoomsTable).Update(updates, "", "").Eq("id", roomID).Execute()
	return err
}

// Update room name in Supabase
func (s *Server) updateRoomName(roomID string, newName string) error {
	updates := map[string]interface{}{
		"name":        newName,
		"last_active": time.Now(),
	}
	_, _, err := s.supabase.From(s.config.SupabaseRoomsTable).Update(updates, "", "").Eq("id", roomID).Execute()
	return err
}

func (s *Server) handleCreateRoom(conn *websocket.Conn, msg models.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Preserva o ID da mensagem original para a resposta
	originalID := msg.MessageID

	if msg.RoomName == "" || msg.Password == "" || msg.PublicKey == "" {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Room name, password, and public key are required"`),
			MessageID: originalID,
		})
		return
	}

	if !passwordRegex.MatchString(msg.Password) {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Password does not match required pattern"`),
			MessageID: originalID,
		})
		return
	}

	// Check if this public key has already created a room
	if existingRoomID, found := s.publicKeys[msg.PublicKey]; found {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(fmt.Sprintf(`"This public key has already created room: %s"`, existingRoomID)),
			MessageID: originalID,
		})
		return
	}

	// Query Supabase for rooms with this public key
	var results []SupabaseRoom
	data, _, err := s.supabase.From(s.config.SupabaseRoomsTable).Select("id", "", false).Eq("public_key", msg.PublicKey).Execute()
	if err == nil {
		err = json.Unmarshal(data, &results)
		if err == nil && len(results) > 0 {
			// Public key already has a room
			conn.WriteJSON(models.Message{
				Type:      "Error",
				Data:      []byte(fmt.Sprintf(`"This public key has already created room: %s"`, results[0].ID)),
				MessageID: originalID,
			})
			return
		}
	}

	roomID := generateRoomID(msg.RoomName)

	// Check if room exists
	_, err = s.fetchRoom(roomID)
	if err == nil {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Room ID conflict"`),
			MessageID: originalID,
		})
		return
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(msg.PublicKey)
	if err != nil {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Invalid public key"`),
			MessageID: originalID,
		})
		return
	}
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Failed to parse public key"`),
			MessageID: originalID,
		})
		return
	}
	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Public key is not RSA"`),
			MessageID: originalID,
		})
		return
	}

	room := ServerRoom{
		Room: models.Room{
			ID:          roomID,
			Name:        msg.RoomName,
			Password:    msg.Password,
			ClientCount: 1, // Creator is the first client
		},
		PublicKey:    rsaPubKey,
		PublicKeyB64: msg.PublicKey,
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
	}

	// Store room in Supabase
	err = s.createRoomInSupabase(room)
	if err != nil {
		log.Printf("Error creating room in Supabase: %v", err)
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Error creating room in database"`),
			MessageID: originalID,
		})
		return
	}

	// Store creator connection and public key mapping
	s.roomCreators[roomID] = conn
	s.clients[conn] = roomID
	s.publicKeys[msg.PublicKey] = roomID // Track that this public key has created a room

	if _, exists := s.networks[roomID]; !exists {
		s.networks[roomID] = []*websocket.Conn{}
	}
	s.networks[roomID] = append(s.networks[roomID], conn)

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		log.Printf("Room %s (%s) created by %s", roomID, msg.RoomName, conn.RemoteAddr().String())
	}

	conn.WriteJSON(models.Message{
		Type:      "RoomCreated",
		RoomID:    roomID,
		RoomName:  msg.RoomName,
		Password:  msg.Password,
		PublicKey: msg.PublicKey,
		MessageID: originalID, // Preserva o ID da mensagem original
	})
}

func (s *Server) handleJoinRoom(conn *websocket.Conn, msg models.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Preserva o ID da mensagem original para a resposta
	originalID := msg.MessageID

	// Fetch room from Supabase
	room, err := s.fetchRoom(msg.RoomID)
	if err != nil {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Room does not exist"`),
			MessageID: originalID,
		})
		return
	}

	if msg.Password != room.Password {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Incorrect password"`),
			MessageID: originalID,
		})
		return
	}

	if msg.PublicKey == "" {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Public key is required"`),
			MessageID: originalID,
		})
		return
	}

	// Check if room is full
	if len(s.networks[msg.RoomID]) >= s.config.MaxClientsPerRoom {
		conn.WriteJSON(models.Message{
			Type:      "Error",
			Data:      []byte(`"Room is full"`),
			MessageID: originalID,
		})
		return
	}

	s.clients[conn] = msg.RoomID
	if _, exists := s.networks[msg.RoomID]; !exists {
		s.networks[msg.RoomID] = []*websocket.Conn{}
	}
	s.networks[msg.RoomID] = append(s.networks[msg.RoomID], conn)
	room.ClientCount = len(s.networks[msg.RoomID])

	// Update last activity time
	err = s.updateRoomActivity(msg.RoomID)
	if err != nil && (s.config.LogLevel == "debug") {
		log.Printf("Error updating room activity: %v", err)
	}

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		log.Printf("Client %s joined room %s", conn.RemoteAddr().String(), msg.RoomID)
	}

	// Envia a resposta ao cliente com o ID da mensagem original
	conn.WriteJSON(models.Message{
		Type:      "RoomJoined",
		RoomID:    msg.RoomID,
		RoomName:  room.Name,
		MessageID: originalID,
	})

	// Notifica os outros participantes da sala sobre o novo cliente
	for _, peer := range s.networks[msg.RoomID] {
		if peer != conn {
			// Informa aos peers que um novo cliente entrou
			peer.WriteJSON(models.Message{
				Type:      "PeerJoined",
				RoomID:    msg.RoomID,
				PublicKey: msg.PublicKey,
				Username:  msg.Username,
			})

			// Informa ao novo cliente sobre os peers existentes
			conn.WriteJSON(models.Message{
				Type:      "PeerJoined",
				RoomID:    msg.RoomID,
				PublicKey: peer.RemoteAddr().String(),
				Username:  "Peer",
			})
		}
	}
}

func (s *Server) handleOffer(conn *websocket.Conn, msg models.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID := s.clients[conn]
	for _, peer := range s.networks[roomID] {
		if peer.RemoteAddr().String() == msg.DestinationID {
			peer.WriteJSON(models.Message{
				Type:          "Offer",
				RoomID:        msg.RoomID,
				PublicKey:     msg.PublicKey,
				DestinationID: msg.DestinationID,
				Offer:         msg.Offer,
			})
			return
		}
	}
}

func (s *Server) handleAnswer(conn *websocket.Conn, msg models.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID := s.clients[conn]
	for _, peer := range s.networks[roomID] {
		if peer.RemoteAddr().String() == msg.DestinationID {
			peer.WriteJSON(models.Message{
				Type:          "Answer",
				RoomID:        msg.RoomID,
				PublicKey:     msg.PublicKey,
				DestinationID: msg.DestinationID,
				Answer:        msg.Answer,
			})
			return
		}
	}
}

func (s *Server) handleCandidate(conn *websocket.Conn, msg models.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID := s.clients[conn]
	for _, peer := range s.networks[roomID] {
		if peer.RemoteAddr().String() == msg.DestinationID {
			peer.WriteJSON(models.Message{
				Type:          "Candidate",
				RoomID:        msg.RoomID,
				PublicKey:     msg.PublicKey,
				DestinationID: msg.DestinationID,
				Candidate:     msg.Candidate,
			})
			return
		}
	}
}

func (s *Server) verifySignature(msg models.Message, room ServerRoom) bool {
	if msg.Signature == "" {
		return false
	}

	sigBytes, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		log.Printf("Failed to decode signature: %v", err)
		return false
	}

	msgCopy := msg
	msgCopy.Signature = ""
	data, err := json.Marshal(msgCopy)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return false
	}
	hash := sha256.Sum256(data)

	err = rsa.VerifyPKCS1v15(room.PublicKey, crypto.SHA256, hash[:], sigBytes)
	return err == nil
}

func (s *Server) handleKick(conn *websocket.Conn, msg models.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.fetchRoom(msg.RoomID)
	if err != nil {
		conn.WriteJSON(models.Message{Type: "Error", Data: []byte(`"Room does not exist"`)})
		return
	}

	if !s.verifySignature(msg, *room) {
		conn.WriteJSON(models.Message{Type: "Error", Data: []byte(`"Invalid signature"`)})
		return
	}

	for _, peer := range s.networks[msg.RoomID] {
		if peer.RemoteAddr().String() == msg.TargetID {
			peer.WriteJSON(models.Message{Type: "Kicked", RoomID: msg.RoomID})
			peer.Close()
			s.removeClient(peer, msg.RoomID)
			log.Printf("Client %s kicked from room %s", msg.TargetID, msg.RoomID)
			conn.WriteJSON(models.Message{Type: "KickSuccess", RoomID: msg.RoomID, TargetID: msg.TargetID})
			return
		}
	}
	conn.WriteJSON(models.Message{Type: "Error", Data: []byte(`"Target client not found"`)})
}

func (s *Server) handleRename(conn *websocket.Conn, msg models.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.fetchRoom(msg.RoomID)
	if err != nil {
		conn.WriteJSON(models.Message{Type: "Error", Data: []byte(`"Room does not exist"`)})
		return
	}

	if !s.verifySignature(msg, *room) {
		conn.WriteJSON(models.Message{Type: "Error", Data: []byte(`"Invalid signature"`)})
		return
	}

	err = s.updateRoomName(msg.RoomID, msg.RoomName)
	if err != nil {
		log.Printf("Error updating room name: %v", err)
		conn.WriteJSON(models.Message{Type: "Error", Data: []byte(`"Error updating room name in database"`)})
		return
	}

	log.Printf("Room %s renamed to %s", msg.RoomID, msg.RoomName)

	for _, peer := range s.networks[msg.RoomID] {
		peer.WriteJSON(models.Message{Type: "RoomRenamed", RoomID: msg.RoomID, RoomName: msg.RoomName})
	}
	conn.WriteJSON(models.Message{Type: "RenameSuccess", RoomID: msg.RoomID, RoomName: msg.RoomName})
}

func (s *Server) handleDelete(conn *websocket.Conn, msg models.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.fetchRoom(msg.RoomID)
	if err != nil {
		conn.WriteJSON(models.Message{Type: "Error", Data: []byte(`"Room does not exist"`)})
		return
	}

	for _, peer := range s.networks[msg.RoomID] {
		peer.WriteJSON(models.Message{Type: "RoomDeleted", RoomID: msg.RoomID})
		peer.Close()
	}

	delete(s.roomCreators, msg.RoomID)
	delete(s.networks, msg.RoomID)
	for c, roomID := range s.clients {
		if roomID == msg.RoomID {
			delete(s.clients, c)
		}
	}

	log.Printf("Room %s deleted", msg.RoomID)
	conn.WriteJSON(models.Message{Type: "DeleteSuccess", RoomID: msg.RoomID})
}

func (s *Server) handleDisconnect(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeClient(conn, s.clients[conn])
}

func (s *Server) removeClient(conn *websocket.Conn, roomID string) {
	if roomID == "" {
		return
	}

	delete(s.clients, conn)

	network := s.networks[roomID]
	for i, peer := range network {
		if peer == conn {
			s.networks[roomID] = append(network[:i], network[i+1:]...)
			break
		}
	}

	if creatorConn, exists := s.roomCreators[roomID]; exists && creatorConn == conn {
		delete(s.roomCreators, roomID)
		err := s.deleteRoomFromSupabase(roomID)
		if err != nil && (s.config.LogLevel == "debug") {
			log.Printf("Error deleting room from Supabase on creator disconnect: %v", err)
		}
	}

	if len(s.networks[roomID]) == 0 {
		delete(s.networks, roomID)
	}

	log.Printf("Client %s left room %s", conn.RemoteAddr().String(), roomID)
}

func (s *Server) deleteStaleRooms() {
	expiryDuration := time.Hour * 24 * time.Duration(s.config.RoomExpiryDays)
	cutoffTime := time.Now().Add(-expiryDuration)

	// Format the cutoffTime to ISO 8601 format which is compatible with Supabase timestamp
	cutoffTimeStr := cutoffTime.Format(time.RFC3339)

	var staleRooms []SupabaseRoom
	data, _, err := s.supabase.From(s.config.SupabaseRoomsTable).Select("*", "", false).Lt("last_active", cutoffTimeStr).Execute()
	if err != nil {
		log.Printf("Error fetching stale rooms: %v", err)
		return
	}

	err = json.Unmarshal(data, &staleRooms)
	if err != nil {
		log.Printf("Error unmarshalling stale rooms data: %v", err)
		return
	}

	for _, room := range staleRooms {
		err := s.deleteRoomFromSupabase(room.ID)
		if err != nil {
			log.Printf("Error deleting stale room %s: %v", room.ID, err)
		} else {
			log.Printf("Deleted stale room %s", room.ID)
		}
	}
}

// RunServer starts the VPN server on the specified port
func RunServer() {
	// Load configuration from environment variables
	serverConfig = loadConfig()

	// Check if Supabase configuration is provided
	if serverConfig.SupabaseURL == "" || serverConfig.SupabaseKey == "" {
		log.Fatal("Supabase URL and API key are required. Please set SUPABASE_URL and SUPABASE_KEY environment variables")
	}

	// Configure WebSocket upgrader
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return serverConfig.AllowAllOrigins
		},
		ReadBufferSize:  serverConfig.ReadBufferSize,
		WriteBufferSize: serverConfig.WriteBufferSize,
	}

	// Compile password regex
	var err error
	passwordRegex, err = regexp.Compile(serverConfig.PasswordPattern)
	if err != nil {
		log.Fatalf("Invalid password pattern '%s': %v", serverConfig.PasswordPattern, err)
	}

	// Create and start the server
	server, err := NewServer(serverConfig)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	http.HandleFunc("/ws", server.handleWebSocket)

	// Add health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Periodically delete stale rooms
	go func() {
		ticker := time.NewTicker(serverConfig.CleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			server.deleteStaleRooms()
		}
	}()

	log.Printf("Server starting on :%s", serverConfig.Port)
	log.Fatal(http.ListenAndServe(":"+serverConfig.Port, nil))
}

func main() {
	RunServer()
}
