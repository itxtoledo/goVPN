package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/itxtoledo/govpn/libs/models"
)

// SignalingServer manages WebSocket connections to the signaling server
type SignalingServer struct {
	ServerURL          string               // WebSocket server URL
	WSConn             *websocket.Conn      // WebSocket connection
	IsConnected        bool                 // Connection status
	MaxRetries         int                  // Maximum number of connection retry attempts
	CurrentRetry       int                  // Current retry attempt counter
	OnConnectionError  func(error)          // Callback for connection errors
	OnMessage          func(models.Message) // Callback for received messages
	OnConnectionStatus func(bool)           // Callback for connection status changes
	OnRawMessage       func([]byte)         // Callback for raw message data
	mu                 sync.RWMutex         // Mutex for thread safety
}

// NewSignalingServer creates a new signaling server connection manager
func NewSignalingServer(serverURL string) *SignalingServer {
	return &SignalingServer{
		ServerURL:    serverURL,
		MaxRetries:   3,
		CurrentRetry: 0,
		IsConnected:  false,
	}
}

// Connect establishes a connection to the signaling server
func (s *SignalingServer) Connect() error {
	u, err := url.Parse(s.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid signaling server URL: %w", err)
	}

	log.Printf("Connecting to signaling server: %s", u.String())
	s.IsConnected = false

	for s.CurrentRetry = 0; s.CurrentRetry < s.MaxRetries; s.CurrentRetry++ {
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("Connection attempt %d failed: %v", s.CurrentRetry+1, err)
			if s.OnConnectionError != nil {
				s.OnConnectionError(err)
			}
			continue
		}

		s.mu.Lock()
		s.WSConn = conn
		s.IsConnected = true
		s.mu.Unlock()

		// Notify of successful connection
		if s.OnConnectionStatus != nil {
			s.OnConnectionStatus(true)
		}

		// Start message handling goroutine
		go s.handleMessages()

		return nil
	}

	if s.OnConnectionStatus != nil {
		s.OnConnectionStatus(false)
	}
	return fmt.Errorf("could not connect to signaling server at %s after %d attempts", s.ServerURL, s.MaxRetries)
}

// Disconnect closes the connection to the signaling server
func (s *SignalingServer) Disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.WSConn != nil {
		s.WSConn.Close()
		s.WSConn = nil
	}
	s.IsConnected = false

	// Notify of disconnection
	if s.OnConnectionStatus != nil {
		s.OnConnectionStatus(false)
	}
}

// handleMessages processes incoming WebSocket messages
func (s *SignalingServer) handleMessages() {
	for {
		s.mu.RLock()
		conn := s.WSConn
		s.mu.RUnlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			s.handleDisconnection()
			return
		}

		// Process raw message if handler exists
		if s.OnRawMessage != nil {
			s.OnRawMessage(message)
		}

		// Process structured message if handler exists
		if s.OnMessage != nil {
			var msg models.Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Error deserializing message: %v", err)
				continue
			}
			s.OnMessage(msg)
		}
	}
}

// handleDisconnection updates the connection state and notifies listeners
func (s *SignalingServer) handleDisconnection() {
	s.mu.Lock()
	s.IsConnected = false
	s.WSConn = nil
	s.mu.Unlock()

	if s.OnConnectionStatus != nil {
		s.OnConnectionStatus(false)
	}
}

// SendMessage sends a message to the signaling server
func (s *SignalingServer) SendMessage(msg interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.WSConn == nil {
		return fmt.Errorf("not connected to signaling server")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	if err := s.WSConn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	return nil
}

// IsConnectionActive returns the current connection state
func (s *SignalingServer) IsConnectionActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsConnected && s.WSConn != nil
}

// SetMaxRetries configures the maximum number of connection attempts
func (s *SignalingServer) SetMaxRetries(retries int) {
	if retries > 0 {
		s.MaxRetries = retries
	}
}
