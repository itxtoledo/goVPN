package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/itxtoledo/govpn/libs/models"
)

// SignalingClient representa uma conexão com o servidor de sinalização
type SignalingClient struct {
	UI             *UIManager
	VPNClient      *VPNClient
	Conn           *websocket.Conn
	ServerAddress  string
	Connected      bool
	LastHeartbeat  time.Time
	MessageHandler func(messageType int, message []byte) error
	PublicKeyStr   string // Public key string to identify this client

	// System to track pending requests by message ID
	pendingRequests     map[string]chan models.SignalingMessage
	pendingRequestsLock sync.Mutex
}

// NewSignalingClient cria uma nova instância do servidor de sinalização
func NewSignalingClient(ui *UIManager, publicKey string) *SignalingClient {
	return &SignalingClient{
		UI:              ui,
		Connected:       false,
		LastHeartbeat:   time.Now(),
		PublicKeyStr:    publicKey,
		pendingRequests: make(map[string]chan models.SignalingMessage),
	}
}

// SetVPNClient sets the reference to the VPNClient for key access
func (s *SignalingClient) SetVPNClient(client *VPNClient) {
	s.VPNClient = client
}

// Connect conecta ao servidor de sinalização
func (s *SignalingClient) Connect(serverAddress string) error {
	if s.Connected {
		// Já está conectado
		return nil
	}

	s.ServerAddress = serverAddress

	// Criar URL para conexão WebSocket
	u, err := url.Parse(serverAddress)
	if err != nil {
		log.Printf("Error parsing server address: %v", err)
		return err
	}

	// Ensure path is set to /ws
	if u.Path == "" || u.Path == "/" {
		u.Path = "/ws"
	}

	log.Printf("Conectando ao servidor de sinalização")
	log.Printf("Connecting to WebSocket server at %s", u.String())

	// Configurar headers para o handshake inicial
	headers := make(map[string][]string)
	headers["User-Agent"] = []string{"goVPN-Client/1.0"}

	// Adicionar identificador do cliente usando a chave pública armazenada diretamente
	if s.PublicKeyStr != "" {
		headers["X-Client-ID"] = []string{s.PublicKeyStr}
	} else if s.VPNClient != nil && s.VPNClient.PublicKeyStr != "" {
		// Fallback para manter compatibilidade
		headers["X-Client-ID"] = []string{s.VPNClient.PublicKeyStr}
	}

	// Estabelecer conexão com o servidor WebSocket com retry
	var conn *websocket.Conn
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	// Try to connect up to 3 times
	for attempts := 0; attempts < 3; attempts++ {
		conn, _, err = dialer.Dial(u.String(), headers)
		if err == nil {
			break // Conexão bem-sucedida
		}

		log.Printf("Connection attempt %d failed: %v", attempts+1, err)

		if attempts < 2 {
			log.Printf("Retrying connection in 1 second...")
			time.Sleep(1 * time.Second)
		}
	}

	if err != nil {
		log.Printf("Failed to connect after 3 attempts: %v", err)
		return err
	}

	s.Conn = conn

	// Configurar handler para mensagens recebidas
	go s.listenForMessages()

	// Marcar como conectado
	s.Connected = true
	s.LastHeartbeat = time.Now()

	// Verificar a conexão com um ping inicial
	err = s.sendPing()
	if err != nil {
		log.Printf("Initial ping failed: %v", err)
		s.Disconnect()
		return fmt.Errorf("connected, but initial ping failed: %v", err)
	}

	log.Printf("Successfully connected to signaling server")
	return nil
}

// Disconnect desconecta do servidor de sinalização
func (s *SignalingClient) Disconnect() error {
	if !s.Connected {
		// Já está desconectado
		return nil
	}

	// Fechar a conexão se existir
	if s.Conn != nil {
		err := s.Conn.Close()
		if err != nil {
			return err
		}
		s.Conn = nil
	}

	// Marcar como desconectado
	s.Connected = false

	return nil
}

// SendHeartbeat envia um heartbeat para o servidor
func (s *SignalingClient) SendHeartbeat() error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	// Os heartbeats não são mais usados nesta versão do protocolo
	// Todos os timestamps são controlados pelo servidor

	return nil
}

// sendPackagedMessage empacota e envia mensagem para o backend e espera pela resposta
// Cria BaseRequest com a chave pública do cliente,
// gera ID da mensagem, empacota na struct SignalingMessage e envia via WebSocket
func (s *SignalingClient) sendPackagedMessage(msgType models.MessageType, payload interface{}) (interface{}, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	// Automatically inject public key into BaseRequest if available
	if s.injectPublicKey(payload) {
		log.Printf("Automatically injected public key into payload")
	}

	// Gerar ID da mensagem
	messageID, err := models.GenerateMessageID()
	if err != nil {
		return nil, fmt.Errorf("error generating message ID: %v", err)
	}

	// Register this message ID to track the response
	responseChan := s.registerPendingRequest(messageID)

	// Serializar payload para JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error serializing payload: %v", err)
	}

	// Criar SignalingMessage
	message := models.SignalingMessage{
		ID:      messageID,
		Type:    msgType,
		Payload: payloadBytes,
	}

	// Enviar a mensagem para o servidor
	log.Printf("Sending message of type %s with ID %s", msgType, messageID)
	err = s.Conn.WriteJSON(message)
	if err != nil {
		return nil, fmt.Errorf("error sending message: %v", err)
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		log.Printf("Received response for message ID %s of type %s", messageID, response.Type)

		// Check if response is an error
		if response.Type == models.TypeError {
			var errorPayload map[string]string
			if err := json.Unmarshal(response.Payload, &errorPayload); err == nil {
				if errorMsg, ok := errorPayload["error"]; ok {
					return nil, fmt.Errorf("server error: %s", errorMsg)
				}
			}
			return nil, errors.New("unknown server error")
		}

		// Parse the response payload based on the request type
		parsedResponse, err := s.parseResponse(msgType, response)
		if err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}
		return parsedResponse, nil

	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response to message ID %s", messageID)
	}
}

// parseResponse parses the response payload based on the request type
func (s *SignalingClient) parseResponse(requestType models.MessageType, response models.SignalingMessage) (interface{}, error) {
	switch requestType {
	case models.TypeCreateRoom:
		if response.Type == models.TypeRoomCreated {
			var resp models.CreateRoomResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal create room response: %v", err)
			}
			return resp, nil
		}

	case models.TypeJoinRoom:
		if response.Type == models.TypeRoomJoined {
			var resp models.JoinRoomResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal join room response: %v", err)
			}
			return resp, nil
		}

	case models.TypeLeaveRoom:
		if response.Type == models.TypeLeaveRoom {
			var resp models.LeaveRoomResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal leave room response: %v", err)
			}
			return resp, nil
		}

	case models.TypeKick:
		if response.Type == models.TypeKickSuccess {
			var resp models.KickResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal kick response: %v", err)
			}
			return resp, nil
		}

	case models.TypeRename:
		if response.Type == models.TypeRenameSuccess {
			var resp models.RenameResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal rename response: %v", err)
			}
			return resp, nil
		}

	case models.TypePing:
		// For ping, we just return a simple success message
		return map[string]interface{}{"status": "success"}, nil
	}

	// If we can't determine the response type, return the raw payload as a map
	var genericResponse map[string]interface{}
	if err := json.Unmarshal(response.Payload, &genericResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal generic response: %v", err)
	}
	return genericResponse, nil
}

// CreateRoom cria uma nova sala no servidor
func (s *SignalingClient) CreateRoom(name string, password string) (*models.CreateRoomResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Creating room: %s", name)

	// Criar payload para a requisição
	payload := &models.CreateRoomRequest{
		BaseRequest: models.BaseRequest{},
		RoomName:    name,
		Password:    password,
	}

	// Enviar solicitação de criação de sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeCreateRoom, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.CreateRoomResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// JoinRoom entra em uma sala
func (s *SignalingClient) JoinRoom(roomID string, password string) (*models.JoinRoomResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Joining room: %s", roomID)

	// Obter o nome de usuário das configurações
	username := s.UI.ConfigManager.GetConfig().Username

	// Criar payload para join room
	payload := &models.JoinRoomRequest{
		BaseRequest: models.BaseRequest{},
		RoomID:      roomID,
		Password:    password,
		Username:    username,
	}

	// Enviar solicitação para entrar na sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeJoinRoom, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.JoinRoomResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// LeaveRoom sai de uma sala
func (s *SignalingClient) LeaveRoom(roomID string) (*models.LeaveRoomResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Leaving room: %s", roomID)

	// Criar payload para leave room
	payload := &models.LeaveRoomRequest{
		BaseRequest: models.BaseRequest{},
		RoomID:      roomID,
	}

	// Enviar solicitação para sair da sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeLeaveRoom, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.LeaveRoomResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// RenameRoom renomeia uma sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) RenameRoom(roomID string, newName string) (*models.RenameResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Renaming room %s to %s", roomID, newName)

	// Criar payload para rename room
	payload := &models.RenameRequest{
		BaseRequest: models.BaseRequest{},
		RoomID:      roomID,
		RoomName:    newName,
	}

	// Enviar solicitação para renomear a sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeRename, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.RenameResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// KickUser expulsa um usuário da sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) KickUser(roomID string, targetID string) (*models.KickResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Kicking user %s from room %s", targetID, roomID)

	// Criar payload para kick user
	payload := &models.KickRequest{
		BaseRequest: models.BaseRequest{},
		RoomID:      roomID,
		TargetID:    targetID,
	}

	// Enviar solicitação para expulsar o usuário usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeKick, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.KickResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// SendMessage envia uma mensagem para o servidor
func (s *SignalingClient) SendMessage(messageType string, payload map[string]interface{}) (interface{}, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	// Adicionar a chave pública ao payload
	payload["publicKey"] = s.PublicKeyStr

	// Converter o messageType para o tipo apropriado
	msgType := models.MessageType(messageType)

	// Enviar a mensagem usando a função de empacotamento
	return s.sendPackagedMessage(msgType, payload)
}

// SetMessageHandler define o handler de mensagens
func (s *SignalingClient) SetMessageHandler(handler func(messageType int, message []byte) error) {
	s.MessageHandler = handler
}

// IsConnected retorna se está conectado ao servidor
func (s *SignalingClient) IsConnected() bool {
	return s.Connected
}

// listenForMessages recebe e processa mensagens vindas do servidor
func (s *SignalingClient) listenForMessages() {
	if s.Conn == nil {
		log.Println("Cannot listen for messages: no websocket connection")
		return
	}

	for {
		if !s.Connected || s.Conn == nil {
			log.Println("Connection closed, stopping message listener")
			return
		}

		msgType, message, err := s.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			s.Connected = false
			s.Conn = nil
			return
		}

		// Process the message here before passing to custom handler
		var sigMsg models.SignalingMessage
		err = json.Unmarshal(message, &sigMsg)
		if err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		log.Printf("Received message of type %s with ID %s", sigMsg.Type, sigMsg.ID)

		// First check if this is a response to a pending request
		// If it is, handlePendingResponse will deliver it to the waiting goroutine
		if s.handlePendingResponse(sigMsg) {
			// Message was delivered to a waiting request handler
			// We can skip further processing
			continue
		}

		// Handle specific message types (for notifications and unsolicited messages)
		switch sigMsg.Type {
		case models.TypeError:
			// Decode error message from base64
			var errorPayload map[string]string
			if err := json.Unmarshal(sigMsg.Payload, &errorPayload); err == nil {
				if errorMsg, ok := errorPayload["error"]; ok {
					log.Printf("Server error: %s", errorMsg)
					// You could notify the UI here
					if s.UI != nil {
						// Example: display error in UI
						//s.UI.ShowErrorNotification(errorMsg)
					}
				}
			} else {
				log.Printf("Failed to decode error payload: %v", err)
			}

		case models.TypeRoomJoined:
			var response models.JoinRoomResponse
			if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
				log.Printf("Failed to unmarshal room joined response: %v", err)
			} else {
				log.Printf("Successfully joined room: %s (%s)", response.RoomName, response.RoomID)
				// Update UI if needed
				if s.UI != nil {
					// Handle room joined in UI
				}
			}

		case models.TypeRoomCreated:
			var response models.CreateRoomResponse
			if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
				log.Printf("Failed to unmarshal room created response: %v", err)
			} else {
				log.Printf("Successfully created room: %s (%s)", response.RoomName, response.RoomID)
				// Update UI if needed
				if s.UI != nil {
					// Handle room created in UI
				}
			}

		case models.TypeLeaveRoom:
			var response models.LeaveRoomResponse
			if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
				log.Printf("Failed to unmarshal leave room response: %v", err)
			} else {
				log.Printf("Successfully left room: %s", response.RoomID)
				// Handle client leaving room
				if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
					// Clean up network connections if needed
				}
			}

		case models.TypeKicked:
			var notification models.KickedNotification
			if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
				log.Printf("Failed to unmarshal kicked notification: %v", err)
			} else {
				reason := notification.Reason
				if reason == "" {
					reason = "No reason provided"
				}
				log.Printf("Kicked from room %s: %s", notification.RoomID, reason)
				// Handle being kicked
				if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
					// Clean up network connections
				}
			}

		case models.TypePeerJoined:
			var notification models.PeerJoinedNotification
			if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
				log.Printf("Failed to unmarshal peer joined notification: %v", err)
			} else {
				username := notification.Username
				if username == "" {
					username = "Unknown user"
				}
				log.Printf("New peer joined room %s: %s (Key: %s)", notification.RoomID, username, notification.PublicKey)
				// Handle new peer
				if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
					// Setup peer connection if needed
				}
			}

		case models.TypePeerLeft:
			var notification models.PeerLeftNotification
			if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
				log.Printf("Failed to unmarshal peer left notification: %v", err)
			} else {
				log.Printf("Peer left room %s: %s", notification.RoomID, notification.PublicKey)
				// Handle peer leaving
				if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
					// Clean up peer connection if needed
				}
			}

		case models.TypeRoomDeleted:
			var notification models.RoomDeletedNotification
			if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
				log.Printf("Failed to unmarshal room deleted notification: %v", err)
			} else {
				log.Printf("Room deleted: %s", notification.RoomID)
				s.VPNClient.NetworkManager.HandleRoomDeleted(notification.RoomID)
			}
		}

		// Also pass to custom handler if configured
		if s.MessageHandler != nil {
			if err := s.MessageHandler(msgType, message); err != nil {
				log.Printf("Error in custom message handler: %v", err)
			}
		}
	}
}

// sendPing envia um ping para verificar a conexão
func (s *SignalingClient) sendPing() error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	log.Printf("Sending ping to server")

	// Create a simple ping message
	pingMessage := map[string]interface{}{
		"action":    "ping",
		"timestamp": time.Now().UnixNano(),
		"publicKey": s.PublicKeyStr,
	}

	// Use the existing message sending infrastructure
	_, err := s.sendPackagedMessage(models.TypePing, pingMessage)
	return err
}

// registerPendingRequest registers a message ID and returns a channel to receive the response
func (s *SignalingClient) registerPendingRequest(messageID string) chan models.SignalingMessage {
	s.pendingRequestsLock.Lock()
	defer s.pendingRequestsLock.Unlock()

	// Create channel for this request
	responseChan := make(chan models.SignalingMessage, 1)
	s.pendingRequests[messageID] = responseChan
	return responseChan
}

// handlePendingResponse routes responses to the appropriate waiting goroutine
func (s *SignalingClient) handlePendingResponse(msg models.SignalingMessage) bool {
	messageID := msg.ID
	if messageID == "" {
		return false
	}

	s.pendingRequestsLock.Lock()
	defer s.pendingRequestsLock.Unlock()

	// Check if we have a pending request waiting for this ID
	if ch, exists := s.pendingRequests[messageID]; exists {
		// Send the response
		select {
		case ch <- msg:
			// Successfully delivered
		default:
			// Channel buffer full (shouldn't happen with buffer size 1)
			log.Printf("Warning: response channel buffer full for message ID %s", messageID)
		}

		// Remove from pending requests after a short delay to ensure message delivery
		go func(id string) {
			time.Sleep(100 * time.Millisecond)
			s.pendingRequestsLock.Lock()
			delete(s.pendingRequests, id)
			s.pendingRequestsLock.Unlock()
		}(messageID)

		return true
	}

	return false
}

// injectPublicKey attempts to inject the public key into any payload struct that has BaseRequest
func (s *SignalingClient) injectPublicKey(payload interface{}) bool {
	// Skip if no public key available
	if s.PublicKeyStr == "" {
		return false
	}

	// Use reflection to check if the payload has a BaseRequest field
	val := reflect.ValueOf(payload)

	// Check if payload is a pointer and not nil
	if val.Kind() != reflect.Ptr || val.IsNil() {
		// Handle special case for map[string]interface{}
		if mapVal, ok := payload.(map[string]interface{}); ok {
			mapVal["publicKey"] = s.PublicKeyStr
			return true
		}
		return false
	}

	// Get the actual value the pointer points to
	val = val.Elem()

	// If it's a struct, look for BaseRequest field
	if val.Kind() == reflect.Struct {
		// Try to find and modify BaseRequest field
		baseReqField := val.FieldByName("BaseRequest")
		if baseReqField.IsValid() && baseReqField.CanSet() {
			// Find the PublicKey field within BaseRequest
			pubKeyField := baseReqField.FieldByName("PublicKey")
			if pubKeyField.IsValid() && pubKeyField.CanSet() && pubKeyField.Kind() == reflect.String {
				// Set the PublicKey field
				pubKeyField.SetString(s.PublicKeyStr)
				return true
			}
		}
	}

	return false
}
