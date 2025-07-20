package signaling

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

// MessageType defines the type of messages that can be sent between client and server
type MessageType string

// Message type constants
const (
	// Client to server message types
	TypeCreateNetwork       MessageType = "CreateNetwork"
	TypeJoinNetwork         MessageType = "JoinNetwork"
	TypeConnectNetwork      MessageType = "ConnectNetwork"
	TypeDisconnectNetwork   MessageType = "DisconnectNetwork"
	TypeLeaveNetwork        MessageType = "LeaveNetwork"
	TypeKick                MessageType = "Kick"
	TypeRename              MessageType = "Rename"
	TypePing                MessageType = "Ping"
	TypeGetComputerNetworks MessageType = "GetComputerNetworks"

	// Server to client message types
	TypeError                MessageType = "Error"
	TypeNetworkCreated       MessageType = "NetworkCreated"
	TypeNetworkJoined        MessageType = "NetworkJoined"
	TypeNetworkConnected     MessageType = "NetworkConnected"
	TypeNetworkDisconnected  MessageType = "NetworkDisconnected"
	TypeNetworkDeleted       MessageType = "NetworkDeleted"
	TypeNetworkRenamed       MessageType = "NetworkRenamed"
	TypeComputerJoined       MessageType = "ComputerJoined"
	TypeComputerLeft         MessageType = "ComputerLeft"
	TypeComputerConnected    MessageType = "ComputerConnected"
	TypeComputerDisconnected MessageType = "ComputerDisconnected"
	TypeKicked               MessageType = "Kicked"
	TypeKickSuccess          MessageType = "KickSuccess"
	TypeRenameSuccess        MessageType = "RenameSuccess"
	TypeDeleteSuccess        MessageType = "DeleteSuccess"
	TypeServerShutdown       MessageType = "ServerShutdown"
	TypeComputerNetworks     MessageType = "ComputerNetworks"
	TypeClientIPInfo         MessageType = "ClientIPInfo"
)

// SignalingMessage represents the wrapper structure for WebSocket communication
type SignalingMessage struct {
	ID      string      `json:"message_id"`
	Type    MessageType `json:"type"`
	Payload []byte      `json:"payload"`
}

// BaseRequest contains common fields used in all messages from client to server
type BaseRequest struct {
	PublicKey string `json:"public_key"` // Base64-encoded Ed25519 public key
}

// ErrorResponse is sent when an error occurs
type ErrorResponse struct {
	Error string `json:"error"`
}

// Event-specific request structs

// CreateNetworkRequest represents a request to create a new network
type CreateNetworkRequest struct {
	BaseRequest
	NetworkName string `json:"network_name"`
	PIN    string `json:"pin"`
}

// CreateNetworkResponse represents a response to a network creation request
type CreateNetworkResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	PIN    string `json:"pin"`
	PublicKey   string `json:"public_key"`
	PeerIP      string `json:"peer_ip"`
}

// JoinNetworkRequest represents a request to join an existing network
type JoinNetworkRequest struct {
	BaseRequest
	NetworkID    string `json:"network_id"`
	PIN     string `json:"pin"`
	ComputerName string `json:"computername,omitempty"`
}

// JoinNetworkResponse represents a response to a network join request
type JoinNetworkResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	PeerIP      string `json:"peer_ip"`
}

// ConnectNetworkRequest represents a request to connect to a previously joined network
type ConnectNetworkRequest struct {
	BaseRequest
	NetworkID    string `json:"network_id"`
	ComputerName string `json:"computername,omitempty"`
}

// ConnectNetworkResponse represents a response to a network connection request
type ConnectNetworkResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	PeerIP      string `json:"peer_ip"`
}

// DisconnectNetworkRequest represents a request to disconnect from a network (but stay joined)
type DisconnectNetworkRequest struct {
	BaseRequest
	NetworkID string `json:"network_id"`
}

// DisconnectNetworkResponse represents a response to a network disconnect request
type DisconnectNetworkResponse struct {
	NetworkID string `json:"network_id"`
}

// LeaveNetworkRequest represents a request to leave a network
type LeaveNetworkRequest struct {
	BaseRequest
	NetworkID string `json:"network_id"`
}

// LeaveNetworkResponse confirms a client has left a network
type LeaveNetworkResponse struct {
	NetworkID string `json:"network_id"`
}

// Network management structs

// KickRequest represents a request to kick a computer from a network
type KickRequest struct {
	BaseRequest
	NetworkID string `json:"network_id"`
	TargetID  string `json:"target_id"`
}

// KickResponse confirms a computer has been kicked
type KickResponse struct {
	NetworkID string `json:"network_id"`
	TargetID  string `json:"target_id"`
}

// RenameRequest represents a request to rename a network
type RenameRequest struct {
	BaseRequest
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
}

// RenameResponse confirms a network has been renamed
type RenameResponse struct {
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
}

// Computer notification structs

// ComputerJoinedNotification notifies that a computer has joined the network
type ComputerJoinedNotification struct {
	NetworkID    string `json:"network_id"`
	PublicKey    string `json:"public_key"`
	ComputerName string `json:"computername,omitempty"`
	PeerIP       string `json:"peer_ip,omitempty"`
}

// ComputerLeftNotification notifies that a computer has left the network
type ComputerLeftNotification struct {
	NetworkID string `json:"network_id"`
	PublicKey string `json:"public_key"`
}

// ComputerConnectedNotification notifies that a computer has connected to the network
type ComputerConnectedNotification struct {
	NetworkID    string `json:"network_id"`
	PublicKey    string `json:"public_key"`
	ComputerName string `json:"computername,omitempty"`
	PeerIP       string `json:"peer_ip,omitempty"`
}

// ComputerDisconnectedNotification notifies that a computer has disconnected from the network (but not left)
type ComputerDisconnectedNotification struct {
	NetworkID string `json:"network_id"`
	PublicKey string `json:"public_key"`
}

// NetworkDeletedNotification notifies that a network has been deleted
type NetworkDeletedNotification struct {
	NetworkID string `json:"network_id"`
}

// KickedNotification notifies a computer they've been kicked
type KickedNotification struct {
	NetworkID string `json:"network_id"`
	Reason    string `json:"reason,omitempty"`
}

// ServerShutdownNotification notifies clients that the server is shutting down
type ServerShutdownNotification struct {
	Message     string `json:"message"`
	ShutdownIn  int    `json:"shutdown_in_seconds"` // Seconds until server shutdown
	RestartInfo string `json:"restart_info,omitempty"`
}

// GetComputerNetworksRequest represents a request to get all networks a computer has joined
type GetComputerNetworksRequest struct {
	BaseRequest
}

// ComputerNetworkInfo represents information about a network a computer has joined
type ComputerNetworkInfo struct {
	NetworkID     string    `json:"network_id"`
	NetworkName   string    `json:"network_name"`
	JoinedAt      time.Time `json:"joined_at"`
	LastConnected time.Time `json:"last_connected"`
}

// ComputerNetworksResponse represents a response containing all networks a computer has joined
type ComputerNetworksResponse struct {
	Networks []ComputerNetworkInfo `json:"networks"`
}

// ClientIPInfoResponse represents client IP address information
type ClientIPInfoResponse struct {
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
}

// Computer represents a computer connected to a network
type Computer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	OwnerID  string `json:"owner_id"`
	IsOnline bool   `json:"is_online"`
	PeerIP   string `json:"peer_ip,omitempty"`
}

// SignalingMessageHandler define o tipo de função para lidar com mensagens de sinalização recebidas.
type SignalingMessageHandler func(messageType MessageType, payload []byte)

// SignalingClient representa uma conexão com o servidor de sinalização
type SignalingClient struct {
	Conn           *websocket.Conn
	ServerAddress  string
	Connected      bool
	LastHeartbeat  time.Time
	MessageHandler SignalingMessageHandler // Usar o novo tipo de função
	PublicKeyStr   string                  // Public key string to identify this client

	// System to track pending requests by message ID
	pendingRequests     map[string]chan SignalingMessage
	pendingRequestsLock sync.Mutex
}

// NewSignalingClient cria uma nova instância do servidor de sinalização
func NewSignalingClient(publicKey string, handler SignalingMessageHandler) *SignalingClient {
	return &SignalingClient{
		Connected:       false,
		LastHeartbeat:   time.Now(),
		PublicKeyStr:    publicKey,
		MessageHandler:  handler, // Atribuir o handler passado
		pendingRequests: make(map[string]chan SignalingMessage),
	}
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
	headers["Computer-Agent"] = []string{"goVPN-Client/1.0"}

	// Adicionar identificador do cliente usando a chave pública armazenada diretamente
	if s.PublicKeyStr != "" {
		headers["X-Client-ID"] = []string{s.PublicKeyStr}
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

// sendPackagedMessage empacota e envia mensagem para o backend e espera pela resposta
// Cria BaseRequest com a chave pública do cliente,
// gera ID da mensagem, empacota na struct SignalingMessage e envia via WebSocket
func (s *SignalingClient) sendPackagedMessage(msgType MessageType, payload interface{}) (interface{}, error) {
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
	message := SignalingMessage{
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
		if response.Type == TypeError {
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
func (s *SignalingClient) parseResponse(requestType MessageType, response SignalingMessage) (interface{}, error) {
	switch requestType {
	case TypeCreateNetwork:
		if response.Type == TypeNetworkCreated {
			var resp CreateNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal create network response: %v", err)
			}
			return resp, nil
		}

	case TypeJoinNetwork:
		if response.Type == TypeNetworkJoined {
			var resp JoinNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal join network response: %v", err)
			}
			return resp, nil
		}

	case TypeConnectNetwork:
		if response.Type == TypeNetworkConnected {
			var resp ConnectNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal connect network response: %v", err)
			}
			return resp, nil
		}

	case TypeDisconnectNetwork:
		if response.Type == TypeDisconnectNetwork || response.Type == TypeNetworkDisconnected {
			var resp DisconnectNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				// If it fails, it might be because the server is using a different format
				// Try to extract just the networkID
				var networkData map[string]interface{}
				if jsonErr := json.Unmarshal(response.Payload, &networkData); jsonErr != nil {
					return nil, fmt.Errorf("failed to unmarshal disconnect network response: %v", err)
				}

				// Extract network ID from the map and return it
				if networkID, ok := networkData["network_id"].(string); ok {
					return map[string]interface{}{
						"network_id": networkID,
					}, nil
				}
				return nil, fmt.Errorf("failed to find network_id in response: %v", err)
			}
			return resp, nil
		}

	case TypeLeaveNetwork:
		if response.Type == TypeLeaveNetwork {
			var resp LeaveNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal leave network response: %v", err)
			}
			return resp, nil
		}

	case TypeKick:
		if response.Type == TypeKickSuccess {
			var resp KickResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal kick response: %v", err)
			}
			return resp, nil
		}

	case TypeRename:
		if response.Type == TypeRenameSuccess {
			var resp RenameResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal rename response: %v", err)
			}
			return resp, nil
		}

	case TypeGetComputerNetworks:
		if response.Type == TypeComputerNetworks {
			var resp ComputerNetworksResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal computer networks response: %v", err)
			}
			return resp, nil
		}

	case TypePing:
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

// CreateNetwork cria uma nova sala no servidor
func (s *SignalingClient) CreateNetwork(name string, pin string) (*CreateNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Creating network: %s", name)

	// Criar payload para a requisição
	payload := &CreateNetworkRequest{
		BaseRequest: BaseRequest{},
		NetworkName: name,
		PIN:    pin,
	}

	// Enviar solicitação de criação de sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(TypeCreateNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(CreateNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// JoinNetwork entra em uma sala
func (s *SignalingClient) JoinNetwork(networkID string, pin string, computername string) (*JoinNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Joining network: %s", networkID)

	// Criar payload para join network
	payload := &JoinNetworkRequest{
		BaseRequest:  BaseRequest{},
		NetworkID:    networkID,
		PIN:     pin,
		ComputerName: computername,
	}

	// Enviar solicitação para entrar na sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(TypeJoinNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(JoinNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// ConnectNetwork conecta a uma sala previamente associada
func (s *SignalingClient) ConnectNetwork(networkID string, computerName string) (*ConnectNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Connecting to network: %s", networkID)

	// Criar payload para connect network
	payload := &ConnectNetworkRequest{
		BaseRequest:  BaseRequest{},
		NetworkID:    networkID,
		ComputerName: computerName,
	}

	// Enviar solicitação para conectar à sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(TypeConnectNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(ConnectNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// DisconnectNetwork desconecta de uma sala sem sair dela
func (s *SignalingClient) DisconnectNetwork(networkID string) (*DisconnectNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Disconnecting from network: %s", networkID)

	// Criar payload para disconnect network
	payload := &DisconnectNetworkRequest{
		BaseRequest: BaseRequest{},
		NetworkID:   networkID,
	}

	// Enviar solicitação para desconectar da sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(TypeDisconnectNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	// Check if we got a map response (like from TypeNetworkDisconnected)
	if respMap, ok := response.(map[string]interface{}); ok {
		// Extract network ID from the map
		networkID, _ := respMap["network_id"].(string)
		resp := DisconnectNetworkResponse{
			NetworkID: networkID,
		}
		return &resp, nil
	} else if resp, ok := response.(DisconnectNetworkResponse); ok {
		return &resp, nil
	}

	// For debugging the response type
	log.Printf("Unexpected response type: %T", response)
	return nil, errors.New("unexpected response type")
}

// LeaveNetwork sai de uma sala
func (s *SignalingClient) LeaveNetwork(networkID string) (*LeaveNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Leaving network: %s", networkID)

	// Criar payload para leave network
	payload := &LeaveNetworkRequest{
		BaseRequest: BaseRequest{},
		NetworkID:   networkID,
	}

	// Enviar solicitação para sair da sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(TypeLeaveNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(LeaveNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// RenameNetwork renomeia uma sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) RenameNetwork(networkID string, newName string) (*RenameResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Renaming network %s to %s", networkID, newName)

	// Criar payload para rename network
	payload := &RenameRequest{
		BaseRequest: BaseRequest{},
		NetworkID:   networkID,
		NetworkName: newName,
	}

	// Enviar solicitação para renomear a sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(TypeRename, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(RenameResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// KickComputer expulsa um usuário da sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) KickComputer(networkID string, targetID string) (*KickResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Kicking computer %s from network %s", targetID, networkID)

	// Criar payload para kick computer
	payload := &KickRequest{
		BaseRequest: BaseRequest{},
		NetworkID:   networkID,
		TargetID:    targetID,
	}

	// Enviar solicitação para expulsar o usuário usando a função de empacotamento
	response, err := s.sendPackagedMessage(TypeKick, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(KickResponse); ok {
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
	msgType := MessageType(messageType)

	// Enviar a mensagem usando a função de empacotamento
	return s.sendPackagedMessage(msgType, payload)
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

		_, message, err := s.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			s.Connected = false
			s.Conn = nil
			return
		}

		// Process the message here before passing to custom handler
		var sigMsg SignalingMessage
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
		case TypeError:
			{
				// Decode error message from base64
				var errorPayload map[string]string
				if err := json.Unmarshal(sigMsg.Payload, &errorPayload); err == nil {
					if errorMsg, ok := errorPayload["error"]; ok {
						log.Printf("Server error: %s", errorMsg)
						// Notify the handler about the error
						if s.MessageHandler != nil {
							s.MessageHandler(TypeError, sigMsg.Payload)
						}
					}
				} else {
					log.Printf("Failed to decode error payload: %v", err)
				}
			}

		case TypeNetworkDisconnected:
			{
				var response DisconnectNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal network disconnected response: %v", err)
				} else {
					log.Printf("Successfully disconnected from network: %s", response.NetworkID)
					// Notify the handler about the network disconnection
					if s.MessageHandler != nil {
						s.MessageHandler(TypeNetworkDisconnected, sigMsg.Payload)
					}
				}
			}

		case TypeNetworkJoined:
			{
				var response JoinNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal network joined response: %v", err)
				} else {
					log.Printf("Successfully joined network: %s (%s)", response.NetworkName, response.NetworkID)
					// Notify the handler about the network joined event
					if s.MessageHandler != nil {
						s.MessageHandler(TypeNetworkJoined, sigMsg.Payload)
					}
				}
			}

		case TypeNetworkCreated:
			{
				var response CreateNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal network created response: %v", err)
				} else {
					log.Printf("Successfully created network: %s (%s)", response.NetworkName, response.NetworkID)
					// Notify the handler about the network created event
					if s.MessageHandler != nil {
						s.MessageHandler(TypeNetworkCreated, sigMsg.Payload)
					}
				}
			}

		case TypeLeaveNetwork:
			{
				var response LeaveNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal leave network response: %v", err)
				} else {
					log.Printf("Successfully left network: %s", response.NetworkID)
					// Notify the handler about the network left event
					if s.MessageHandler != nil {
						s.MessageHandler(TypeLeaveNetwork, sigMsg.Payload)
					}
				}
			}

		case TypeKicked:
			{
				var notification KickedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal kicked notification: %v", err)
				} else {
					reason := notification.Reason
					if reason == "" {
						reason = "No reason provided"
					}
					log.Printf("Kicked from network %s: %s", notification.NetworkID, reason)
					// Notify the handler about the kicked event
					if s.MessageHandler != nil {
						s.MessageHandler(TypeKicked, sigMsg.Payload)
					}
				}
			}

		case TypeComputerJoined:
			{
				var notification ComputerJoinedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal computer joined notification: %v", err)
				} else {
					computerName := notification.ComputerName
					if computerName == "" {
						computerName = "Unknown computer"
					}
					log.Printf("New computer joined network %s: %s (Key: %s, IP: %s)", notification.NetworkID, computerName, notification.PublicKey, notification.PeerIP)

					// Notify the handler about the computer joined event
					if s.MessageHandler != nil {
						s.MessageHandler(TypeComputerJoined, sigMsg.Payload)
					}
				}
			}

		case TypeComputerConnected:
			{
				var notification ComputerConnectedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal computer connected notification: %v", err)
				} else {
					computerName := notification.ComputerName
					if computerName == "" {
						computerName = "Unknown computer"
					}
					log.Printf("Computer connected to network %s: %s (Key: %s, IP: %s)", notification.NetworkID, computerName, notification.PublicKey, notification.PeerIP)

					// Notify the handler about the computer connected event
					if s.MessageHandler != nil {
						s.MessageHandler(TypeComputerConnected, sigMsg.Payload)
					}
				}
			}

		case TypeComputerLeft:
			var notification ComputerLeftNotification
			if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
				log.Printf("Failed to unmarshal computer left notification: %v", err)
			} else {
				log.Printf("Computer left network %s: %s", notification.NetworkID, notification.PublicKey)

				// Notify the handler about the computer left event
				if s.MessageHandler != nil {
					s.MessageHandler(TypeComputerLeft, sigMsg.Payload)
				}
			}

		case TypeNetworkDeleted:
			{
				var notification NetworkDeletedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal network deleted notification: %v", err)
				} else {
					log.Printf("Network deleted: %s", notification.NetworkID)
					// Notify the handler about the network deleted event
					if s.MessageHandler != nil {
						s.MessageHandler(TypeNetworkDeleted, sigMsg.Payload)
					}
				}
			}

		case TypeComputerNetworks:
			{
				var response ComputerNetworksResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal computer networks response: %v", err)
				} else {
					log.Printf("========== COMPUTER NETWORKS RECEIVED ==========")
					log.Printf("Total networks found: %d", len(response.Networks))

					for i, network := range response.Networks {
						log.Printf("Network %d: %s (ID: %s)", i+1, network.NetworkName, network.NetworkID)
						log.Printf("  Joined at: %s", network.JoinedAt.Format(time.RFC1123))
						log.Printf("  Last connected: %s", network.LastConnected.Format(time.RFC1123))
						log.Printf("  ---")
					}

					// Notify the handler about the updated network list
					if s.MessageHandler != nil {
						s.MessageHandler(TypeComputerNetworks, sigMsg.Payload)
					}
				}
			}

		case TypeClientIPInfo:
			{
				var ipInfo ClientIPInfoResponse
				if err := json.Unmarshal(sigMsg.Payload, &ipInfo); err != nil {
					log.Printf("Failed to unmarshal client IP info: %v", err)
				} else {
					log.Printf("Received client IP info - IPv4: %s, IPv6: %s", ipInfo.IPv4, ipInfo.IPv6)

					// Notify the handler about the IP info
					if s.MessageHandler != nil {
						s.MessageHandler(TypeClientIPInfo, sigMsg.Payload)
					}
				}
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
	pingMessage := map[string]interface{}{}
	pingMessage["action"] = "ping"
	pingMessage["timestamp"] = time.Now().UnixNano()
	pingMessage["publicKey"] = s.PublicKeyStr

	// Use the existing message sending infrastructure
	_, err := s.sendPackagedMessage(TypePing, pingMessage)
	return err
}

// registerPendingRequest registers a message ID and returns a channel to receive the response
func (s *SignalingClient) registerPendingRequest(messageID string) chan SignalingMessage {
	s.pendingRequestsLock.Lock()
	defer s.pendingRequestsLock.Unlock()

	// Create channel for this request
	responseChan := make(chan SignalingMessage, 1)
	s.pendingRequests[messageID] = responseChan
	return responseChan
}

// handlePendingResponse routes responses to the appropriate waiting goroutine
func (s *SignalingClient) handlePendingResponse(msg SignalingMessage) bool {
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

// GetComputerNetworks requests all networks the computer has joined from the server
func (s *SignalingClient) GetComputerNetworks() (*ComputerNetworksResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Requesting computer networks from server")

	// Create payload for the request
	payload := &GetComputerNetworksRequest{
		BaseRequest: BaseRequest{},
	}

	// Send request using the packaging function
	response, err := s.sendPackagedMessage(TypeGetComputerNetworks, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(ComputerNetworksResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}