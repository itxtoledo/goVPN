package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"golang.org/x/crypto/pbkdf2"
	_ "github.com/mattn/go-sqlite3"
)

const (
	serverAddr       = "ws://localhost:8080/ws" // Ajuste após deploy
	roomsDB          = "rooms.db"
	stunServer       = "stun:stun.l.google.com:19302"
	turnServer       = "turn:turn.example.com:3478" // Substitua pelo seu TURN server
	turnUsername     = "username"                   // Substitua pelo seu TURN username
	turnPassword     = "password"                   // Substitua pelo seu TURN password
)

var passwordRegex = regexp.MustCompile(`^\d{4}$`)

// Room represents a virtual room
type Room struct {
	ID         string
	Name       string
	Password   string
	Status     string
	IsCreator  bool
	PrivateKey string
	PublicKey  string
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
	Candidate      json.RawMessage `json:"candidate,omitempty"`
	Offer          json.RawMessage `json:"offer,omitempty"`
	Answer         json.RawMessage `json:"answer,omitempty"`
	ClientID       string          `json:"clientID,omitempty"`
	DestinationID  string          `json:"destinationID,omitempty"`
}

// Client represents a client in the network
type Client struct {
	ID            string
	Conn          *websocket.Conn
	PeerConns     map[string]*webrtc.PeerConnection
	DataChannels  map[string]*webrtc.DataChannel
	Send          chan []byte
	NetworkID     string
	NATStatus     string
}

// VPNClient manages the VPN client
type VPNClient struct {
	Client      *Client
	Networks    map[string][]string // RoomID -> ClientIDs
	Conn        *websocket.Conn
	DB          *sql.DB
	App         fyne.App
	Window      fyne.Window
	StatusLabel *widget.Label
	NATLabel    *widget.Label
	Rooms       []Room
	RoomList    *widget.List
	mu          sync.Mutex
}

// NewVPNClient creates a new VPN client
func NewVPNClient() *VPNClient {
	a := app.New()
	a.Settings().SetTheme(&gamingTheme{})
	w := a.NewWindow("GoVPN - Virtual LAN para Jogos")
	w.Resize(fyne.NewSize(600, 500))

	clientID := generateID()
	client := &Client{
		ID:           clientID,
		Send:         make(chan []byte),
		PeerConns:    make(map[string]*webrtc.PeerConnection),
		DataChannels: make(map[string]*webrtc.DataChannel),
		NATStatus:    "Desconhecido",
	}

	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	vpn := &VPNClient{
		Client:   client,
		Networks: make(map[string][]string),
		App:      a,
		Window:   w,
		DB:       db,
		Rooms:    loadRooms(db),
	}
	vpn.StatusLabel = widget.NewLabel("Status: Desconectado")
	vpn.NATLabel = widget.NewLabel("NAT Traversal: Desconhecido")
	return vpn
}

// gamingTheme defines the red gamer theme
type gamingTheme struct{}

func (t *gamingTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) fyne.ThemeColor {
	switch name {
	case theme.ColorNameBackground:
		return fyne.NewColor(0x00, 0x00, 0x00, 0xFF)
	case theme.ColorNameButton:
		return fyne.NewColor(0xFF, 0x00, 0x00, 0xFF)
	case theme.ColorNameDisabled:
		return fyne.NewColor(0xCC, 0x00, 0x00, 0x80)
	case theme.ColorNameForeground:
		return fyne.NewColor(0xFF, 0xFF, 0xFF, 0xFF)
	case theme.ColorNamePrimary:
		return fyne.NewColor(0xFF, 0x00, 0x00, 0xFF)
	case theme.ColorNameHover:
		return fyne.NewColor(0xCC, 0x00, 0x00, 0xFF)
	case theme.ColorNameInputBackground:
		return fyne.NewColor(0x1C, 0x25, 0x26, 0xFF)
	case theme.ColorNamePlaceHolder:
		return fyne.NewColor(0x80, 0x80, 0x80, 0xFF)
	default:
		return theme.DefaultTheme().Color(name, fyne.ThemeVariantDark)
	}
}

func (t *gamingTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *gamingTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *gamingTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 20
	default:
		return theme.DefaultTheme().Size(name)
	}
}

// generateID creates a unique ID
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateRSAKeys generates an RSA key pair
func generateRSAKeys() (privateKey *rsa.PrivateKey, publicKey string, err error) {
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", err
	}
	publicKey = base64.StdEncoding.EncodeToString(pubKeyBytes)
	return privateKey, publicKey, nil
}

// signMessage signs a message with the private key
func signMessage(msg Message, privateKey *rsa.PrivateKey) (string, error) {
	msgCopy := msg
	msgCopy.Signature = ""
	data, err := json.Marshal(msgCopy)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// initDB initializes the SQLite database
func initDB() (*sql.DB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %v", err)
	}
	configDir := filepath.Join(homeDir, ".govpn")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %v", err)
	}
	dbPath := filepath.Join(configDir, roomsDB)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS rooms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			password TEXT NOT NULL,
			status TEXT NOT NULL,
			is_creator BOOLEAN NOT NULL,
			private_key TEXT NOT NULL,
			public_key TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create rooms table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			description TEXT
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create config table: %v", err)
	}

	// Populate default configuration values if they don't exist
	if err := initDefaultConfig(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize default configuration: %v", err)
	}

	return db, nil
}

// initDefaultConfig inserts default configuration values if they don't exist
func initDefaultConfig(db *sql.DB) error {
	defaults := []struct {
		key         string
		value       string
		description string
	}{
		{"server_address", "ws://localhost:8080/ws", "WebSocket server address for signaling"},
		{"stun_server", "stun:stun.l.google.com:19302", "STUN server for NAT traversal"},
		{"turn_server", "turn:turn.example.com:3478", "TURN server for relay connections"},
		{"turn_username", "username", "TURN server username"},
		{"turn_password", "password", "TURN server password"},
		{"ice_timeout_seconds", "30", "ICE gathering timeout in seconds"},
		{"max_retries", "5", "Maximum number of reconnection attempts"},
		{"reconnect_delay_seconds", "5", "Delay between reconnection attempts in seconds"},
	}

	for _, cfg := range defaults {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM config WHERE key = ?", cfg.key).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check if config exists: %v", err)
		}

		if count == 0 {
			_, err = db.Exec(
				"INSERT INTO config (key, value, description) VALUES (?, ?, ?)",
				cfg.key, cfg.value, cfg.description,
			)
			if err != nil {
				return fmt.Errorf("failed to insert default config %s: %v", cfg.key, err)
			}
		}
	}

	return nil
}

// getConfig retrieves a configuration value from the database
func (c *VPNClient) getConfig(key string) (string, error) {
	var value string
	err := c.DB.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("failed to get config %s: %v", key, err)
	}
	return value, nil
}

// getConfigWithDefault retrieves a configuration value or returns a default if not found
func (c *VPNClient) getConfigWithDefault(key, defaultValue string) string {
	value, err := c.getConfig(key)
	if err != nil {
		log.Printf("Warning: Using default value for %s: %v", key, err)
		return defaultValue
	}
	return value
}

// updateConfig updates a configuration value in the database
func (c *VPNClient) updateConfig(key, value, description string) error {
	_, err := c.DB.Exec(
		"UPDATE config SET value = ?, description = ? WHERE key = ?",
		value, description, key,
	)
	if err != nil {
		return fmt.Errorf("failed to update config %s: %v", key, err)
	}
	return nil
}

// loadRooms loads rooms from SQLite
func loadRooms(db *sql.DB) []Room {
	rows, err := db.Query("SELECT id, name, password, status, is_creator, private_key, public_key FROM rooms")
	if err != nil {
		log.Printf("Failed to query rooms: %v", err)
		return []Room{}
	}
	defer rows.Close()

	var rooms []Room
	for rows.Next() {
		var room Room
		var isCreator int
		if err := rows.Scan(&room.ID, &room.Name, &room.Password, &room.Status, &isCreator, &room.PrivateKey, &room.PublicKey); err != nil {
			log.Printf("Failed to scan room: %v", err)
			continue
		}
		room.IsCreator = isCreator != 0
		rooms = append(rooms, room)
	}
	return rooms
}

// saveRoom saves a room to SQLite
func (c *VPNClient) saveRoom(room Room) error {
	_, err := c.DB.Exec(`
		INSERT OR REPLACE INTO rooms (id, name, password, status, is_creator, private_key, public_key)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, room.ID, room.Name, room.Password, room.Status, room.IsCreator, room.PrivateKey, room.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to save room: %v", err)
	}
	return nil
}

// deleteRoom deletes a room from SQLite
func (c *VPNClient) deleteRoom(roomID string) error {
	_, err := c.DB.Exec("DELETE FROM rooms WHERE id = ?", roomID)
	if err != nil {
		return fmt.Errorf("failed to delete room: %v", err)
	}
	return nil
}

// updateRoomStatus updates a room's status in SQLite
func (c *VPNClient) updateRoomStatus(roomID, status string) error {
	_, err := c.DB.Exec("UPDATE rooms SET status = ? WHERE id = ?", status, roomID)
	if err != nil {
		return fmt.Errorf("failed to update room status: %v", err)
	}
	return nil
}

// checkRoomStatus checks if a room is online
func (c *VPNClient) checkRoomStatus(roomID string) string {
	if c.Conn == nil {
		return "offline"
	}
	err := c.Conn.WriteJSON(Message{Type: "JoinRoom", RoomID: roomID, ClientID: c.Client.ID})
	if err != nil {
		return "offline"
	}
	if _, exists := c.Networks[roomID]; exists && len(c.Networks[roomID]) > 1 {
		return "online"
	}
	return "offline"
}

// connectToServer connects to the WebSocket server
func (c *VPNClient) connectToServer() error {
	// Get server address and retry parameters from configuration
	serverAddr := c.getConfigWithDefault("server_address", serverAddr)
	maxRetriesStr := c.getConfigWithDefault("max_retries", "5")
	reconnectDelayStr := c.getConfigWithDefault("reconnect_delay_seconds", "5")
	
	// Parse retry parameters
	maxRetries := 5
	fmt.Sscanf(maxRetriesStr, "%d", &maxRetries)
	
	reconnectDelay := 5
	fmt.Sscanf(reconnectDelayStr, "%d", &reconnectDelay)

	var conn *websocket.Conn
	var err error
	for retries := 0; retries < maxRetries; retries++ {
		log.Printf("Connecting to server at %s (attempt %d of %d)", serverAddr, retries+1, maxRetries)
		conn, _, err = websocket.DefaultDialer.Dial(serverAddr, nil)
		if err == nil {
			c.Conn = conn
			c.Client.Conn = conn
			log.Printf("Successfully connected to server")
			return nil
		}
		log.Printf("Failed to connect to server (attempt %d): %v", retries+1, err)
		delay := time.Second * time.Duration(reconnectDelay*(retries+1))
		log.Printf("Retrying in %v...", delay)
		time.Sleep(delay)
	}
	return fmt.Errorf("failed to connect to server after %d retries: %v", maxRetries, err)
}

// setupWebRTC configures WebRTC for a room
func (c *VPNClient) setupWebRTC(room Room) error {
	// Get WebRTC configuration values from database
	stunServerURL := c.getConfigWithDefault("stun_server", stunServer)
	turnServerURL := c.getConfigWithDefault("turn_server", turnServer)
	turnUser := c.getConfigWithDefault("turn_username", turnUsername)
	turnPass := c.getConfigWithDefault("turn_password", turnPassword)
	iceTimeoutStr := c.getConfigWithDefault("ice_timeout_seconds", "30")
	
	// Parse ICE timeout
	iceTimeout := 30
	fmt.Sscanf(iceTimeoutStr, "%d", &iceTimeout)

	// Configure WebRTC with values from the database
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{stunServerURL}},
			{
				URLs:       []string{turnServerURL},
				Username:   turnUser,
				Credential: turnPass,
			},
		},
		ICETransportPolicy: webrtc.ICETransportPolicyAll,
	}

	// Set ICE gathering timeout
	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetICETimeouts(time.Duration(iceTimeout)*time.Second, time.Duration(iceTimeout)*time.Second)
	api := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))
	
	// Create peer connection with the configured API
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %v", err)
	}

	dataChannel, err := peerConnection.CreateDataChannel("vpn", nil)
	if err != nil {
		return fmt.Errorf("failed to create data channel: %v", err)
	}

	c.Client.PeerConns[room.ID] = peerConnection
	c.Client.DataChannels[room.ID] = dataChannel

	dataChannel.OnOpen(func() {
		log.Printf("Data channel opened for room %s", room.ID)
		c.Client.NATStatus = "Conectado (P2P)"
		c.NATLabel.SetText("NAT Traversal: Conectado (P2P)")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		key := pbkdf2.Key([]byte(room.Password), []byte(room.ID), 4096, 32, sha256.New)
		data, err := decrypt(msg.Data, key)
		if err != nil {
			log.Printf("Error decrypting data: %v", err)
			return
		}
		log.Printf("Received data: %s", string(data))
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			candidateJSON, err := json.Marshal(candidate.ToJSON())
			if err != nil {
				log.Printf("Error marshaling ICE candidate: %v", err)
				return
			}
			for _, peerID := range c.Networks[room.ID] {
				if peerID != c.Client.ID {
					c.Conn.WriteJSON(Message{
						Type:          "Candidate",
						RoomID:        room.ID,
						ClientID:      c.Client.ID,
						DestinationID: peerID,
						Candidate:     candidateJSON,
					})
				}
			}
		}
	})

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Peer connection state: %s", state)
		if state == webrtc.PeerConnectionStateConnected {
			c.Client.NATStatus = "Conectado (P2P)"
			c.NATLabel.SetText("NAT Traversal: Conectado (P2P)")
		} else if state == webrtc.PeerConnectionStateFailed {
			c.Client.NATStatus = "Falhou (tentando TURN)"
			c.NATLabel.SetText("NAT Traversal: Falhou (tentando TURN)")
		}
	})

	return nil
}

// connectToRoom connects to an existing room
func (c *VPNClient) connectToRoom(room Room) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.setupWebRTC(room); err != nil {
		return err
	}

	err := c.Conn.WriteJSON(Message{
		Type:     "JoinRoom",
		RoomID:   room.ID,
		Password: room.Password,
		ClientID: c.Client.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to join room: %v", err)
	}

	c.Client.NetworkID = room.ID
	if _, exists := c.Networks[room.ID]; !exists {
		c.Networks[room.ID] = []string{}
	}
	c.Networks[room.ID] = append(c.Networks[room.ID], c.Client.ID)

	c.StatusLabel.SetText(fmt.Sprintf("Status: Conectado à sala %s", room.Name))
	go c.handleWebSocket()
	return nil
}

// disconnectFromRoom disconnects from the current room
func (c *VPNClient) disconnectFromRoom() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, pc := range c.Client.PeerConns {
		pc.Close()
	}
	c.Client.PeerConns = make(map[string]*webrtc.PeerConnection)
	c.Client.DataChannels = make(map[string]*webrtc.DataChannel)
	if c.Client.NetworkID != "" {
		delete(c.Networks, c.Client.NetworkID)
		c.Client.NetworkID = ""
	}
	c.Client.NATStatus = "Desconhecido"
	c.NATLabel.SetText("NAT Traversal: Desconhecido")
	c.StatusLabel.SetText("Status: Desconectado")
}

// handleWebSocket manages WebSocket messages
func (c *VPNClient) handleWebSocket() {
	for {
		var msg Message
		err := c.Client.Conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			c.disconnectFromRoom()
			return
		}

		switch msg.Type {
		case "RoomCreated":
			// Handled during room creation
		case "RoomRenamed":
			for i, room := range c.Rooms {
				if room.ID == msg.RoomID {
					c.Rooms[i].Name = msg.RoomName
					if err := c.saveRoom(c.Rooms[i]); err != nil {
						log.Printf("Failed to save room: %v", err)
					}
					c.RoomList.Refresh()
					c.StatusLabel.SetText(fmt.Sprintf("Status: Sala renomeada para %s", msg.RoomName))
					break
				}
			}
		case "Kicked":
			c.disconnectFromRoom()
			dialog.ShowInformation("Expulso", "Você foi expulso da sala.", c.Window)
			c.RoomList.Refresh()
		case "RoomDeleted":
			c.disconnectFromRoom()
			for i, room := range c.Rooms {
				if room.ID == msg.RoomID {
					if err := c.deleteRoom(room.ID); err != nil {
						log.Printf("Failed to delete room: %v", err)
					}
					c.Rooms = append(c.Rooms[:i], c.Rooms[i+1:]...)
					break
				}
			}
			c.RoomList.Refresh()
			dialog.ShowInformation("Sala Deletada", "A sala foi deletada pelo criador.", c.Window)
		case "PeerJoined":
			if msg.ClientID != c.Client.ID {
				c.Networks[c.Client.NetworkID] = append(c.Networks[c.Client.NetworkID], msg.ClientID)
				go c.initiateWebRTCOffer(msg.ClientID)
			}
		case "Offer":
			go c.handleWebRTCOffer(msg)
		case "Answer":
			go c.handleWebRTCAnswer(msg)
		case "Candidate":
			go c.handleWebRTCCandidate(msg)
		case "Error":
			var errMsg string
			json.Unmarshal(msg.Data, &errMsg)
			c.StatusLabel.SetText(fmt.Sprintf("Status: Erro: %s", errMsg))
		}
	}
}

// initiateWebRTCOffer creates a WebRTC offer for a peer
func (c *VPNClient) initiateWebRTCOffer(peerID string) {
	peerConnection := c.Client.PeerConns[c.Client.NetworkID]
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Printf("Error creating offer: %v", err)
		return
	}

	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		log.Printf("Error setting local description: %v", err)
		return
	}

	offerJSON, err := json.Marshal(offer)
	if err != nil {
		log.Printf("Error marshaling offer: %v", err)
		return
	}

	c.Conn.WriteJSON(Message{
		Type:          "Offer",
		RoomID:        c.Client.NetworkID,
		ClientID:      c.Client.ID,
		DestinationID: peerID,
		Offer:         offerJSON,
	})
}

// handleWebRTCOffer processes a WebRTC offer
func (c *VPNClient) handleWebRTCOffer(msg Message) {
	peerConnection, exists := c.Client.PeerConns[msg.RoomID]
	if !exists {
		return
	}

	var offer webrtc.SessionDescription
	if err := json.Unmarshal(msg.Offer, &offer); err != nil {
		log.Printf("Error unmarshaling offer: %v", err)
		return
	}

	err := peerConnection.SetRemoteDescription(offer)
	if err != nil {
		log.Printf("Error setting remote description: %v", err)
		return
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Printf("Error creating answer: %v", err)
		return
	}

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		log.Printf("Error setting local description: %v", err)
		return
	}

	answerJSON, err := json.Marshal(answer)
	if err != nil {
		log.Printf("Error marshaling answer: %v", err)
		return
	}

	c.Conn.WriteJSON(Message{
		Type:          "Answer",
		RoomID:        msg.RoomID,
		ClientID:      c.Client.ID,
		DestinationID: msg.ClientID,
		Answer:        answerJSON,
	})
}

// handleWebRTCAnswer processes a WebRTC answer
func (c *VPNClient) handleWebRTCAnswer(msg Message) {
	peerConnection, exists := c.Client.PeerConns[msg.RoomID]
	if !exists {
		return
	}

	var answer webrtc.SessionDescription
	if err := json.Unmarshal(msg.Answer, &answer); err != nil {
		log.Printf("Error unmarshaling answer: %v", err)
		return
	}

	err := peerConnection.SetRemoteDescription(answer)
	if err != nil {
		log.Printf("Error setting remote description: %v", err)
	}
}

// handleWebRTCCandidate processes an ICE candidate
func (c *VPNClient) handleWebRTCCandidate(msg Message) {
	peerConnection, exists := c.Client.PeerConns[msg.RoomID]
	if !exists {
		return
	}

	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal(msg.Candidate, &candidate); err != nil {
		log.Printf("Error unmarshaling candidate: %v", err)
		return
	}

	err := peerConnection.AddICECandidate(candidate)
	if err != nil {
		log.Printf("Error adding ICE candidate: %v", err)
	}
}

// encrypt encrypts data with AES
func encrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)
	return ciphertext, nil
}

// decrypt decrypts data with AES
func decrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(data) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]
	plaintext := make([]byte, len(ciphertext))

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}

// showAboutWindow displays the About window
func (c *VPNClient) showAboutWindow() {
	aboutWindow := c.App.NewWindow("Sobre o GoVPN")
	aboutWindow.Resize(fyne.NewSize(400, 300))

	title := canvas.NewText("Sobre o GoVPN", color.RGBA{0xFF, 0x00, 0x00, 0xFF})
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 18
	title.Alignment = fyne.TextAlignCenter

	description := widget.NewLabel(
		"GoVPN é uma Virtual LAN (VLAN) para jogos, projetada para conectar jogadores como se estivessem na mesma rede local. Suporta NAT traversal com STUN/TURN para conexões P2P robustas. Crie ou junte-se a salas de jogo para reduzir latência e desfrutar de partidas multiplayer fluidas. Dados das salas são armazenados localmente em SQLite.",
	)
	description.Wrapping = fyne.TextWrapWord

	closeButton := widget.NewButton("Fechar", func() {
		aboutWindow.Close()
	})

	content := container.NewVBox(
		title,
		description,
		closeButton,
	)

	panel := container.NewBorder(
		nil,
		nil,
		canvas.NewRectangle(color.RGBA{0xCC, 0x00, 0x00, 0xFF}),
		canvas.NewRectangle(color.RGBA{0xCC, 0x00, 0x00, 0xFF}),
		content,
	)
	panel.Objects[0].(*canvas.Rectangle).SetMinSize(fyne.NewSize(5, 0))
	panel.Objects[1].(*canvas.Rectangle).SetMinSize(fyne.NewSize(5, 0))

	aboutWindow.SetContent(container.NewPadded(panel))
	aboutWindow.Show()
}

// showConfigWindow displays the configuration window
func (c *VPNClient) showConfigWindow() {
	configWindow := c.App.NewWindow("Configurações")
	configWindow.Resize(fyne.NewSize(500, 400))

	title := canvas.NewText("Configurações do GoVPN", color.RGBA{0xFF, 0x00, 0x00, 0xFF})
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 18
	title.Alignment = fyne.TextAlignCenter

	// Get current configuration values
	serverAddress, _ := c.getConfig("server_address")
	stunServer, _ := c.getConfig("stun_server")
	turnServer, _ := c.getConfig("turn_server")
	turnUsername, _ := c.getConfig("turn_username")
	turnPassword, _ := c.getConfig("turn_password")
	iceTimeout, _ := c.getConfig("ice_timeout_seconds")
	maxRetries, _ := c.getConfig("max_retries")
	reconnectDelay, _ := c.getConfig("reconnect_delay_seconds")

	// Create form entries
	serverAddressEntry := widget.NewEntry()
	serverAddressEntry.SetText(serverAddress)
	
	stunServerEntry := widget.NewEntry()
	stunServerEntry.SetText(stunServer)
	
	turnServerEntry := widget.NewEntry()
	turnServerEntry.SetText(turnServer)
	
	turnUsernameEntry := widget.NewEntry()
	turnUsernameEntry.SetText(turnUsername)
	
	turnPasswordEntry := widget.NewPasswordEntry()
	turnPasswordEntry.SetText(turnPassword)
	
	iceTimeoutEntry := widget.NewEntry()
	iceTimeoutEntry.SetText(iceTimeout)
	
	maxRetriesEntry := widget.NewEntry()
	maxRetriesEntry.SetText(maxRetries)
	
	reconnectDelayEntry := widget.NewEntry()
	reconnectDelayEntry.SetText(reconnectDelay)

	// Create the form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Servidor WebSocket", Widget: serverAddressEntry},
			{Text: "Servidor STUN", Widget: stunServerEntry},
			{Text: "Servidor TURN", Widget: turnServerEntry},
			{Text: "Usuário TURN", Widget: turnUsernameEntry},
			{Text: "Senha TURN", Widget: turnPasswordEntry},
			{Text: "Timeout ICE (segundos)", Widget: iceTimeoutEntry},
			{Text: "Tentativas de Reconexão", Widget: maxRetriesEntry},
			{Text: "Atraso entre Tentativas (segundos)", Widget: reconnectDelayEntry},
		},
		OnSubmit: func() {
			// Update configuration values in the database
			c.updateConfig("server_address", serverAddressEntry.Text, "WebSocket server address for signaling")
			c.updateConfig("stun_server", stunServerEntry.Text, "STUN server for NAT traversal")
			c.updateConfig("turn_server", turnServerEntry.Text, "TURN server for relay connections")
			c.updateConfig("turn_username", turnUsernameEntry.Text, "TURN server username")
			c.updateConfig("turn_password", turnPasswordEntry.Text, "TURN server password")
			c.updateConfig("ice_timeout_seconds", iceTimeoutEntry.Text, "ICE gathering timeout in seconds")
			c.updateConfig("max_retries", maxRetriesEntry.Text, "Maximum number of reconnection attempts")
			c.updateConfig("reconnect_delay_seconds", reconnectDelayEntry.Text, "Delay between reconnection attempts in seconds")
			
			dialog.ShowInformation("Sucesso", "Configurações salvas com sucesso. As alterações terão efeito na próxima inicialização.", configWindow)
		},
		OnCancel: func() {
			configWindow.Close()
		},
		SubmitText: "Salvar",
		CancelText: "Cancelar",
	}

	resetButton := widget.NewButton("Restaurar Padrões", func() {
		dialog.ShowConfirm("Restaurar Padrões", "Tem certeza que deseja restaurar todas as configurações para os valores padrão?", func(confirmed bool) {
			if confirmed {
				// Reset all config values to defaults
				c.DB.Exec("DELETE FROM config")
				initDefaultConfig(c.DB)
				
				// Update the form fields
				serverAddressEntry.SetText("ws://localhost:8080/ws")
				stunServerEntry.SetText("stun:stun.l.google.com:19302")
				turnServerEntry.SetText("turn:turn.example.com:3478")
				turnUsernameEntry.SetText("username")
				turnPasswordEntry.SetText("password")
				iceTimeoutEntry.SetText("30")
				maxRetriesEntry.SetText("5")
				reconnectDelayEntry.SetText("5")
				
				dialog.ShowInformation("Sucesso", "Configurações restauradas para os valores padrão.", configWindow)
			}
		}, configWindow)
	})

	content := container.NewVBox(
		title,
		form,
		resetButton,
	)

	panel := container.NewBorder(
		nil,
		nil,
		canvas.NewRectangle(color.RGBA{0xCC, 0x00, 0x00, 0xFF}),
		canvas.NewRectangle(color.RGBA{0xCC, 0x00, 0x00, 0xFF}),
		content,
	)
	panel.Objects[0].(*canvas.Rectangle).SetMinSize(fyne.NewSize(5, 0))
	panel.Objects[1].(*canvas.Rectangle).SetMinSize(fyne.NewSize(5, 0))

	configWindow.SetContent(container.NewPadded(panel))
	configWindow.Show()
}

// setupGUI configures the graphical interface
func (c *VPNClient) setupGUI() {
	title := canvas.NewText("GoVPN - Virtual LAN para Jogos", color.RGBA{0xFF, 0x00, 0x00, 0xFF})
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 20
	title.Alignment = fyne.TextAlignCenter

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Nome da Sala")
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Senha (4 dígitos)")
	roomIDLabel := widget.NewLabel("ID da Sala: (Gerado pelo servidor após criação)")

	addRoomButton := widget.NewButton("Criar Sala", func() {
		if nameEntry.Text == "" || passwordEntry.Text == "" {
			c.StatusLabel.SetText("Status: Nome e senha são obrigatórios")
			return
		}
		if !passwordRegex.MatchString(passwordEntry.Text) {
			c.StatusLabel.SetText("Status: A senha deve ser um número de 4 dígitos")
			return
		}

		privateKey, publicKey, err := generateRSAKeys()
		if err != nil {
			c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao gerar chaves RSA: %v", err))
			return
		}
		privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao serializar chave privada: %v", err))
			return
		}

		msg := Message{
			Type:      "CreateRoom",
			RoomName:  nameEntry.Text,
			Password:  passwordEntry.Text,
			PublicKey: publicKey,
			ClientID:  c.Client.ID,
		}
		err = c.Conn.WriteJSON(msg)
		if err != nil {
			c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao criar sala: %v", err))
			return
		}

		var response Message
		err = c.Conn.ReadJSON(&response)
		if err != nil {
			c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao receber resposta do servidor: %v", err))
			return
		}
		if response.Type != "RoomCreated" {
			var errMsg string
			json.Unmarshal(response.Data, &errMsg)
			c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao criar sala: %s", errMsg))
			return
		}

		room := Room{
			ID:         response.RoomID,
			Name:       response.RoomName,
			Password:   response.Password,
			Status:     c.checkRoomStatus(response.RoomID),
			IsCreator:  true,
			PrivateKey: base64.StdEncoding.EncodeToString(privateKeyBytes),
			PublicKey:  publicKey,
		}
		c.Rooms = append(c.Rooms, room)
		if err := c.saveRoom(room); err != nil {
			c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao salvar sala: %v", err))
			return
		}
		c.RoomList.Refresh()
		c.StatusLabel.SetText("Status: Sala criada com sucesso")
		roomIDLabel.SetText(fmt.Sprintf("ID da Sala: %s", room.ID))
		nameEntry.SetText("")
		passwordEntry.SetText("")
	})

	aboutButton := widget.NewButton("Sobre", func() {
		c.showAboutWindow()
	})

	configButton := widget.NewButton("Configurações", func() {
		c.showConfigWindow()
	})

	// Create a row of buttons
	buttonRow := container.NewHBox(
		addRoomButton,
		configButton,
		aboutButton,
	)

	c.RoomList = widget.NewList(
		func() int {
			return len(c.Rooms)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				canvas.NewRectangle(color.RGBA{0xCC, 0x00, 0x00, 0xFF}),
				widget.NewLabel(""),
				widget.NewButton("Conectar", nil),
				widget.NewButton("Remover", nil),
				widget.NewButton("Expulsar", nil),
				widget.NewButton("Renomear", nil),
				widget.NewButton("Deletar", nil),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			room := c.Rooms[id]
			statusIcon := "🔴"
			if room.Status == "online" {
				statusIcon = "🟢"
			}
			item.(*fyne.Container).Objects[0].(*canvas.Rectangle).SetMinSize(fyne.NewSize(5, 30))
			item.(*fyne.Container).Objects[1].(*widget.Label).SetText(fmt.Sprintf("%s (%s)", room.Name, statusIcon))
			item.(*fyne.Container).Objects[1].(*widget.Label).TextStyle = fyne.TextStyle{Bold: true}

			connectButton := item.(*fyne.Container).Objects[2].(*widget.Button)
			if c.Client.NetworkID == room.ID {
				connectButton.SetText("Desconectar")
				connectButton.OnTapped = func() {
					c.disconnectFromRoom()
					room.Status = c.checkRoomStatus(room.ID)
					c.Rooms[id].Status = room.Status
					if err := c.updateRoomStatus(room.ID, room.Status); err != nil {
						log.Printf("Failed to update room status: %v", err)
					}
					c.RoomList.Refresh()
				}
			} else {
				connectButton.SetText("Conectar")
				connectButton.OnTapped = func() {
					if err := c.connectToRoom(room); err != nil {
						c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao conectar à sala: %v", err))
						return
					}
					room.Status = c.checkRoomStatus(room.ID)
					c.Rooms[id].Status = room.Status
					if err := c.updateRoomStatus(room.ID, room.Status); err != nil {
						log.Printf("Failed to update room status: %v", err)
					}
					c.RoomList.Refresh()
				}
			}

			item.(*fyne.Container).Objects[3].(*widget.Button).OnTapped = func() {
				if err := c.deleteRoom(room.ID); err != nil {
					log.Printf("Failed to delete room: %v", err)
				}
				c.Rooms = append(c.Rooms[:id], c.Rooms[id+1:]...)
				c.RoomList.Refresh()
			}

			kickButton := item.(*fyne.Container).Objects[4].(*widget.Button)
			renameButton := item.(*fyne.Container).Objects[5].(*widget.Button)
			deleteButton := item.(*fyne.Container).Objects[6].(*widget.Button)
			if room.IsCreator {
				kickButton.Show()
				renameButton.Show()
				deleteButton.Show()

				kickButton.OnTapped = func() {
					targetIDEntry := widget.NewEntry()
					targetIDEntry.SetPlaceHolder("Endereço do Cliente (ex.: 192.168.1.1:1234)")
					dialog.ShowForm("Expulsar Cliente", "Expulsar", "Cancelar", []*widget.FormItem{
						{Text: "Endereço", Widget: targetIDEntry},
					}, func(confirm bool) {
						if confirm && targetIDEntry.Text != "" {
							privateKeyBytes, err := base64.StdEncoding.DecodeString(room.PrivateKey)
							if err != nil {
								c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao decodificar chave privada: %v", err))
								return
							}
							privateKey, err := x509.ParsePKCS8PrivateKey(privateKeyBytes)
							if err != nil {
								c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao parsear chave privada: %v", err))
								return
							}
							msg := Message{Type: "Kick", RoomID: room.ID, TargetID: targetIDEntry.Text}
							msg.Signature, err = signMessage(msg, privateKey.(*rsa.PrivateKey))
							if err != nil {
								c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao assinar mensagem: %v", err))
								return
							}
							c.Conn.WriteJSON(msg)
						}
					}, c.Window)
				}

				renameButton.OnTapped = func() {
					newNameEntry := widget.NewEntry()
					newNameEntry.SetPlaceHolder("Novo Nome da Sala")
					dialog.ShowForm("Renomear Sala", "Renomear", "Cancelar", []*widget.FormItem{
						{Text: "Novo Nome", Widget: newNameEntry},
					}, func(confirm bool) {
						if confirm && newNameEntry.Text != "" {
							privateKeyBytes, err := base64.StdEncoding.DecodeString(room.PrivateKey)
							if err != nil {
								c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao decodificar chave privada: %v", err))
								return
							}
							privateKey, err := x509.ParsePKCS8PrivateKey(privateKeyBytes)
							if err != nil {
								c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao parsear chave privada: %v", err))
								return
							}
							msg := Message{Type: "Rename", RoomID: room.ID, RoomName: newNameEntry.Text}
							msg.Signature, err = signMessage(msg, privateKey.(*rsa.PrivateKey))
							if err != nil {
								c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao assinar mensagem: %v", err))
								return
							}
							c.Conn.WriteJSON(msg)
						}
					}, c.Window)
				}

				deleteButton.OnTapped = func() {
					dialog.ShowConfirm("Deletar Sala", "Tem certeza que deseja deletar a sala?", func(confirm bool) {
						if confirm {
							privateKeyBytes, err := base64.StdEncoding.DecodeString(room.PrivateKey)
							if err != nil {
								c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao decodificar chave privada: %v", err))
								return
							}
							privateKey, err := x509.ParsePKCS8PrivateKey(privateKeyBytes)
							if err != nil {
								c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao parsear chave privada: %v", err))
								return
							}
							msg := Message{Type: "Delete", RoomID: room.ID}
							msg.Signature, err = signMessage(msg, privateKey.(*rsa.PrivateKey))
							if err != nil {
								c.StatusLabel.SetText(fmt.Sprintf("Status: Erro ao assinar mensagem: %v", err))
								return
							}
							c.Conn.WriteJSON(msg)
						}
					}, c.Window)
				}
			} else {
				kickButton.Hide()
				renameButton.Hide()
				deleteButton.Hide()
			}
		},
	)

	content := container.NewVBox(
		title,
		widget.NewLabel("Adicionar Nova Sala de Jogo"),
		nameEntry,
		passwordEntry,
		roomIDLabel,
		buttonRow, // Use the row of buttons instead of individual buttons
		widget.NewLabel("Salas de Jogo"),
		c.RoomList,
		c.StatusLabel,
		c.NATLabel,
	)

	panel := container.NewBorder(
		nil,
		nil,
		canvas.NewRectangle(color.RGBA{0xCC, 0x00, 0x00, 0xFF}),
		canvas.NewRectangle(color.RGBA{0xCC, 0x00, 0x00, 0xFF}),
		content,
	)
	panel.Objects[0].(*canvas.Rectangle).SetMinSize(fyne.NewSize(5, 0))
	panel.Objects[1].(*canvas.Rectangle).SetMinSize(fyne.NewSize(5, 0))

	c.Window.SetContent(container.NewPadded(panel))
}

// main starts the program
func main() {
	vpn := NewVPNClient()
	defer vpn.DB.Close()

	if err := vpn.connectToServer(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	for i, room := range vpn.Rooms {
		room.Status = vpn.checkRoomStatus(room.ID)
		vpn.Rooms[i] = room
		if err := vpn.updateRoomStatus(room.ID, room.Status); err != nil {
			log.Printf("Failed to update room status: %v", err)
		}
	}

	vpn.setupGUI()
	vpn.Window.ShowAndRun()
}