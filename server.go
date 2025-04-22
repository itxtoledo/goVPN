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
	"github.com/nedpals/supabase-go"
)

// Config structure to hold our server configuration
type Config struct {
	Port                string
	AllowAllOrigins     bool
	PasswordPattern     string
	MaxRooms            int
	MaxClientsPerRoom   int
	LogLevel            string
	IdleTimeout         time.Duration
	PingInterval        time.Duration
	ReadBufferSize      int
	WriteBufferSize     int
	SupabaseURL         string
	SupabaseKey         string
	SupabaseRoomsTable  string
	CleanupInterval     time.Duration
	RoomExpiryDays      int
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	config := Config{
		Port:              getEnv("PORT", "8080"),
		AllowAllOrigins:   getEnvBool("ALLOW_ALL_ORIGINS", true),
		PasswordPattern:   getEnv("PASSWORD_PATTERN", `^\d{4}$`),
		MaxRooms:          getEnvInt("MAX_ROOMS", 100),
		MaxClientsPerRoom: getEnvInt("MAX_CLIENTS_PER_ROOM", 10),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		IdleTimeout:       time.Second * time.Duration(getEnvInt("IDLE_TIMEOUT_SECONDS", 60)),
		PingInterval:      time.Second * time.Duration(getEnvInt("PING_INTERVAL_SECONDS", 30)),
		ReadBufferSize:    getEnvInt("READ_BUFFER_SIZE", 1024),
		WriteBufferSize:   getEnvInt("WRITE_BUFFER_SIZE", 1024),
		SupabaseURL:       getEnv("SUPABASE_URL", ""),
		SupabaseKey:       getEnv("SUPABASE_KEY", ""),
		SupabaseRoomsTable: getEnv("SUPABASE_ROOMS_TABLE", "rooms"),
		CleanupInterval:   time.Hour * time.Duration(getEnvInt("CLEANUP_INTERVAL_HOURS", 24)),
		RoomExpiryDays:    getEnvInt("ROOM_EXPIRY_DAYS", 30),
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

var config Config
var upgrader websocket.Upgrader
var passwordRegex *regexp.Regexp

// Room represents a virtual room
type Room struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Password    string         `json:"password"`
	PublicKey   *rsa.PublicKey `json:"-"` // Not stored in Supabase directly
	PublicKeyB64 string        `json:"public_key"` // Stored as base64 string in Supabase
	CreatedAt   time.Time      `json:"created_at"`
	LastActive  time.Time      `json:"last_active"`
}

// SupabaseRoom is a struct for room data stored in Supabase
type SupabaseRoom struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Password    string    `json:"password"`
	PublicKey   string    `json:"public_key"` // Base64 encoded public key
	CreatedAt   time.Time `json:"created_at"`
	LastActive  time.Time `json:"last_active"`
}

// Message represents a WebSocket message
type Message struct {
	Type           string          `json:"type"`
	RoomID         string          `json:"roomID,omitempty"`
	RoomName       string          `json:"roomName,omitempty"`
	Password       string          `json:"password,omitempty"`
	PublicKey      string          `json:"publicKey,omitempty"`
	TargetID       string          `json:"targetID,omitempty"`
	Signature      string          `json:"signature,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
	Candidate      json.RawMessage `json:"candidate,omitempty"`      // ICE candidate
	Offer          json.RawMessage `json:"offer,omitempty"`         // WebRTC offer
	Answer         json.RawMessage `json:"answer,omitempty"`        // WebRTC answer
	ClientID       string          `json:"clientID,omitempty"`      // Unique client ID
	DestinationID  string          `json:"destinationID,omitempty"` // Target client for signaling
}

// Server manages clients and rooms
type Server struct {
	clients      map[*websocket.Conn]string
	networks     map[string][]*websocket.Conn
	roomCreators map[string]*websocket.Conn
	mu           sync.RWMutex
	config       Config
	supabase     *supabase.Client
}

func NewServer(cfg Config) (*Server, error) {
	if cfg.SupabaseURL == "" || cfg.SupabaseKey == "" {
		return nil, errors.New("Supabase URL and API key are required")
	}

	supaClient := supabase.CreateClient(cfg.SupabaseURL, cfg.SupabaseKey)
	
	return &Server{
		clients:      make(map[*websocket.Conn]string),
		networks:     make(map[string][]*websocket.Conn),
		roomCreators: make(map[string]*websocket.Conn),
		config:       cfg,
		supabase:     &supaClient,
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

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			s.handleDisconnect(conn)
			return
		}

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
		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
}

// Fetch a room from Supabase
func (s *Server) fetchRoom(roomID string) (*Room, error) {
	var results []SupabaseRoom
	err := s.supabase.DB.From(s.config.SupabaseRoomsTable).Select("*").Eq("id", roomID).Execute(&results)
	if err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return nil, errors.New("Room not found")
	}
	
	// Parse the public key from the base64 string
	pubKeyBytes, err := base64.StdEncoding.DecodeString(results[0].PublicKey)
	if err != nil {
		return nil, fmt.Errorf("Invalid public key format: %v", err)
	}
	
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse public key: %v", err)
	}
	
	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("Public key is not RSA")
	}
	
	room := &Room{
		ID:          results[0].ID,
		Name:        results[0].Name,
		Password:    results[0].Password,
		PublicKey:   rsaPubKey,
		PublicKeyB64: results[0].PublicKey,
		CreatedAt:   results[0].CreatedAt,
		LastActive:  results[0].LastActive,
	}
	
	return room, nil
}

// Create a room in Supabase
func (s *Server) createRoomInSupabase(room Room) error {
	supaRoom := SupabaseRoom{
		ID:        room.ID,
		Name:      room.Name,
		Password:  room.Password,
		PublicKey: room.PublicKeyB64,
		CreatedAt: room.CreatedAt,
		LastActive: room.LastActive,
	}
	
	var result SupabaseRoom
	err := s.supabase.DB.From(s.config.SupabaseRoomsTable).Insert(supaRoom).Execute(&result)
	return err
}

// Delete a room from Supabase
func (s *Server) deleteRoomFromSupabase(roomID string) error {
	return s.supabase.DB.From(s.config.SupabaseRoomsTable).Delete().Eq("id", roomID).Execute(nil)
}

// Update room last activity time in Supabase
func (s *Server) updateRoomActivity(roomID string) error {
	updates := map[string]interface{}{
		"last_active": time.Now(),
	}
	return s.supabase.DB.From(s.config.SupabaseRoomsTable).Update(updates).Eq("id", roomID).Execute(nil)
}

// Update room name in Supabase
func (s *Server) updateRoomName(roomID string, newName string) error {
	updates := map[string]interface{}{
		"name": newName,
		"last_active": time.Now(),
	}
	return s.supabase.DB.From(s.config.SupabaseRoomsTable).Update(updates).Eq("id", roomID).Execute(nil)
}

// Count total rooms in Supabase
func (s *Server) countRoomsInSupabase() (int, error) {
	type CountResult struct {
		Count int `json:"count"`
	}
	
	var result []CountResult
	err := s.supabase.DB.From(s.config.SupabaseRoomsTable).Select("count", "exact").Execute(&result)
	if err != nil {
		return 0, err
	}
	
	if len(result) == 0 {
		return 0, errors.New("Failed to get room count")
	}
	
	return result[0].Count, nil
}

func (s *Server) handleCreateRoom(conn *websocket.Conn, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check room limit
	roomCount, err := s.countRoomsInSupabase()
	if err != nil {
		log.Printf("Error counting rooms: %v", err)
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Error accessing database"`)})
		return
	}
	
	if roomCount >= s.config.MaxRooms {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Maximum room limit reached"`)})
		return
	}

	if msg.RoomName == "" || msg.Password == "" || msg.PublicKey == "" || msg.ClientID == "" {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Room name, password, public key, and client ID are required"`)})
		return
	}

	if !passwordRegex.MatchString(msg.Password) {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Password does not match required pattern"`)})
		return
	}

	roomID := generateRoomID(msg.RoomName)
	
	// Check if room exists
	_, err = s.fetchRoom(roomID)
	if err == nil {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Room ID conflict"`)})
		return
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(msg.PublicKey)
	if err != nil {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Invalid public key"`)})
		return
	}
	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Failed to parse public key"`)})
		return
	}
	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Public key is not RSA"`)})
		return
	}

	room := Room{
		ID:          roomID,
		Name:        msg.RoomName,
		Password:    msg.Password,
		PublicKey:   rsaPubKey,
		PublicKeyB64: msg.PublicKey,
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}
	
	// Store room in Supabase
	err = s.createRoomInSupabase(room)
	if err != nil {
		log.Printf("Error creating room in Supabase: %v", err)
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Error creating room in database"`)})
		return
	}
	
	// Store creator connection 
	s.roomCreators[roomID] = conn
	s.clients[conn] = roomID
	if _, exists := s.networks[roomID]; !exists {
		s.networks[roomID] = []*websocket.Conn{}
	}
	s.networks[roomID] = append(s.networks[roomID], conn)

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		log.Printf("Room %s (%s) created by %s", roomID, msg.RoomName, conn.RemoteAddr().String())
	}
	conn.WriteJSON(Message{Type: "RoomCreated", RoomID: roomID, RoomName: msg.RoomName, Password: msg.Password, ClientID: msg.ClientID})
}

func (s *Server) handleJoinRoom(conn *websocket.Conn, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fetch room from Supabase
	room, err := s.fetchRoom(msg.RoomID)
	if err != nil {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Room does not exist"`)})
		return
	}

	if msg.Password != room.Password {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Incorrect password"`)})
		return
	}

	if msg.ClientID == "" {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Client ID is required"`)})
		return
	}
	
	// Check if room is full
	if len(s.networks[msg.RoomID]) >= s.config.MaxClientsPerRoom {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Room is full"`)})
		return
	}

	s.clients[conn] = msg.RoomID
	if _, exists := s.networks[msg.RoomID]; !exists {
		s.networks[msg.RoomID] = []*websocket.Conn{}
	}
	s.networks[msg.RoomID] = append(s.networks[msg.RoomID], conn)
	
	// Update last activity time
	err = s.updateRoomActivity(msg.RoomID)
	if err != nil && (s.config.LogLevel == "debug") {
		log.Printf("Error updating room activity: %v", err)
	}

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		log.Printf("Client %s (%s) joined room %s", conn.RemoteAddr().String(), msg.ClientID, msg.RoomID)
	}

	// Notify peers and share client IDs
	for _, peer := range s.networks[msg.RoomID] {
		if peer != conn {
			peer.WriteJSON(Message{Type: "PeerJoined", RoomID: msg.RoomID, ClientID: msg.ClientID})
			conn.WriteJSON(Message{Type: "PeerJoined", RoomID: msg.RoomID, ClientID: s.clients[peer]})
		}
	}
}

func (s *Server) handleOffer(conn *websocket.Conn, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID := s.clients[conn]
	for _, peer := range s.networks[roomID] {
		if peer.RemoteAddr().String() == msg.DestinationID {
			peer.WriteJSON(Message{
				Type:          "Offer",
				RoomID:        msg.RoomID,
				ClientID:      msg.ClientID,
				DestinationID: msg.DestinationID,
				Offer:         msg.Offer,
			})
			return
		}
	}
}

func (s *Server) handleAnswer(conn *websocket.Conn, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID := s.clients[conn]
	for _, peer := range s.networks[roomID] {
		if peer.RemoteAddr().String() == msg.DestinationID {
			peer.WriteJSON(Message{
				Type:          "Answer",
				RoomID:        msg.RoomID,
				ClientID:      msg.ClientID,
				DestinationID: msg.DestinationID,
				Answer:        msg.Answer,
			})
			return
		}
	}
}

func (s *Server) handleCandidate(conn *websocket.Conn, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID := s.clients[conn]
	for _, peer := range s.networks[roomID] {
		if peer.RemoteAddr().String() == msg.DestinationID {
			peer.WriteJSON(Message{
				Type:          "Candidate",
				RoomID:        msg.RoomID,
				ClientID:      msg.ClientID,
				DestinationID: msg.DestinationID,
				Candidate:     msg.Candidate,
			})
			return
		}
	}
}

func (s *Server) verifySignature(msg Message, room Room) bool {
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

func (s *Server) handleKick(conn *websocket.Conn, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.fetchRoom(msg.RoomID)
	if err != nil {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Room does not exist"`)})
		return
	}

	if !s.verifySignature(msg, *room) {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Invalid signature"`)})
		return
	}

	for _, peer := range s.networks[msg.RoomID] {
		if peer.RemoteAddr().String() == msg.TargetID {
			peer.WriteJSON(Message{Type: "Kicked", RoomID: msg.RoomID})
			peer.Close()
			s.removeClient(peer, msg.RoomID)
			log.Printf("Client %s kicked from room %s", msg.TargetID, msg.RoomID)
			conn.WriteJSON(Message{Type: "KickSuccess", RoomID: msg.RoomID, TargetID: msg.TargetID})
			return
		}
	}
	conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Target client not found"`)})
}

func (s *Server) handleRename(conn *websocket.Conn, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.fetchRoom(msg.RoomID)
	if err != nil {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Room does not exist"`)})
		return
	}

	if !s.verifySignature(msg, *room) {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Invalid signature"`)})
		return
	}

	err = s.updateRoomName(msg.RoomID, msg.RoomName)
	if err != nil {
		log.Printf("Error updating room name: %v", err)
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Error updating room name in database"`)})
		return
	}
	
	log.Printf("Room %s renamed to %s", msg.RoomID, msg.RoomName)

	for _, peer := range s.networks[msg.RoomID] {
		peer.WriteJSON(Message{Type: "RoomRenamed", RoomID: msg.RoomID, RoomName: msg.RoomName})
	}
	conn.WriteJSON(Message{Type: "RenameSuccess", RoomID: msg.RoomID, RoomName: msg.RoomName})
}

func (s *Server) handleDelete(conn *websocket.Conn, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.fetchRoom(msg.RoomID)
	if err != nil {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Room does not exist"`)})
		return
	}

	if !s.verifySignature(msg, *room) {
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Invalid signature"`)})
		return
	}

	err = s.deleteRoomFromSupabase(msg.RoomID)
	if err != nil {
		log.Printf("Error deleting room from Supabase: %v", err)
		conn.WriteJSON(Message{Type: "Error", Data: []byte(`"Error deleting room from database"`)})
		return
	}

	for _, peer := range s.networks[msg.RoomID] {
		peer.WriteJSON(Message{Type: "RoomDeleted", RoomID: msg.RoomID})
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
	conn.WriteJSON(Message{Type: "DeleteSuccess", RoomID: msg.RoomID})
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

	var staleRooms []SupabaseRoom
	err := s.supabase.DB.From(s.config.SupabaseRoomsTable).Select("*").Lt("last_active", cutoffTime).Execute(&staleRooms)
	if err != nil {
		log.Printf("Error fetching stale rooms: %v", err)
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

func main() {
	// Load configuration from environment variables
	config = loadConfig()
	
	// Check if Supabase configuration is provided
	if config.SupabaseURL == "" || config.SupabaseKey == "" {
		log.Fatal("Supabase URL and API key are required. Please set SUPABASE_URL and SUPABASE_KEY environment variables")
	}
	
	// Configure WebSocket upgrader
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { 
			return config.AllowAllOrigins
		},
		ReadBufferSize:  config.ReadBufferSize,
		WriteBufferSize: config.WriteBufferSize,
	}
	
	// Compile password regex
	var err error
	passwordRegex, err = regexp.Compile(config.PasswordPattern)
	if err != nil {
		log.Fatalf("Invalid password pattern '%s': %v", config.PasswordPattern, err)
	}
	
	// Create and start the server
	server, err := NewServer(config)
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
		ticker := time.NewTicker(config.CleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			server.deleteStaleRooms()
		}
	}()
	
	log.Printf("Server starting on :%s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}