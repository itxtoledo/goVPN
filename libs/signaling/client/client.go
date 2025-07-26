package client

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
	signaling_models "github.com/itxtoledo/govpn/libs/signaling/models"
	"github.com/itxtoledo/govpn/libs/utils"
)

// SignalingMessageHandler is a function type used to handle signaling messages received by the client.
// It takes two parameters:
// - messageType: The type of the signaling message, defined by the signaling_models.MessageType enum.
// - payload: The raw message payload as a byte slice, which can be unmarshaled into the appropriate structure.
// This handler is invoked whenever a new signaling message is received, allowing the client to process it.
type SignalingMessageHandler func(messageType signaling_models.MessageType, payload []byte)

// SignalingClient representa uma conexão com o servidor de sinalização
type SignalingClient struct {
	Conn           *websocket.Conn
	ServerAddress  string
	Connected      bool
	LastHeartbeat  time.Time
	MessageHandler SignalingMessageHandler // Usar o novo tipo de função
	PublicKeyStr   string                  // Public key string to identify this client

	// System to track pending requests by message ID
	pendingRequests     map[string]chan signaling_models.SignalingMessage
	pendingRequestsLock sync.Mutex
}

// NewSignalingClient cria uma nova instância do servidor de sinalização
func NewSignalingClient(publicKey string, handler SignalingMessageHandler) *SignalingClient {
	return &SignalingClient{
		Connected:       false,
		LastHeartbeat:   time.Now(),
		PublicKeyStr:    publicKey,
		MessageHandler:  handler, // Assign the passed handler
		pendingRequests: make(map[string]chan signaling_models.SignalingMessage),
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

// sendPackagedMessage packages and sends a message to the backend and waits for a response.
// Creates a BaseRequest with the client's public key,
// generates a message ID, packages it into the SignalingMessage struct, and sends it via WebSocket.
func (s *SignalingClient) sendPackagedMessage(msgType signaling_models.MessageType, payload interface{}) (interface{}, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	// Automatically inject public key into BaseRequest if available
	if s.injectPublicKey(payload) {
		log.Printf("Automatically injected public key into payload")
	}

	// Gerar ID da mensagem
	messageID, err := utils.GenerateMessageID()
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
	message := signaling_models.SignalingMessage{
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
		if response.Type == signaling_models.TypeError {
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
func (s *SignalingClient) parseResponse(requestType signaling_models.MessageType, response signaling_models.SignalingMessage) (interface{}, error) {
	switch requestType {
	case signaling_models.TypeCreateNetwork:
		if response.Type == signaling_models.TypeNetworkCreated {
			var resp signaling_models.CreateNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal create network response: %v", err)
			}
			return resp, nil
		}

	case signaling_models.TypeJoinNetwork:
		if response.Type == signaling_models.TypeNetworkJoined {
			var resp signaling_models.JoinNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal join network response: %v", err)
			}
			return resp, nil
		}

	case signaling_models.TypeConnectNetwork:
		if response.Type == signaling_models.TypeNetworkConnected {
			var resp signaling_models.ConnectNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal connect network response: %v", err)
			}
			return resp, nil
		}

	case signaling_models.TypeDisconnectNetwork:
		if response.Type == signaling_models.TypeDisconnectNetwork || response.Type == signaling_models.TypeNetworkDisconnected {
			var resp signaling_models.DisconnectNetworkResponse
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

	case signaling_models.TypeLeaveNetwork:
		if response.Type == signaling_models.TypeLeaveNetwork {
			var resp signaling_models.LeaveNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal leave network response: %v", err)
			}
			return resp, nil
		}

	case signaling_models.TypeKick:
		if response.Type == signaling_models.TypeKickSuccess {
			var resp signaling_models.KickResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal kick response: %v", err)
			}
			return resp, nil
		}

	case signaling_models.TypeRename:
		if response.Type == signaling_models.TypeRenameSuccess {
			var resp signaling_models.RenameResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal rename response: %v", err)
			}
			return resp, nil
		}

	case signaling_models.TypeGetComputerNetworks:
		if response.Type == signaling_models.TypeComputerNetworks {
			var resp signaling_models.ComputerNetworksResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal computer networks response: %v", err)
			}
			return resp, nil
		}

	case signaling_models.TypePing:
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
func (s *SignalingClient) CreateNetwork(name string, pin string, computerName string) (*signaling_models.CreateNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Creating network: %s", name)

	// Criar payload para a requisição
	payload := &signaling_models.CreateNetworkRequest{
		BaseRequest: signaling_models.BaseRequest{},
		NetworkName: name,
		PIN:         pin,
		ComputerName: computerName,
	}

	// Enviar solicitação de criação de sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(signaling_models.TypeCreateNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(signaling_models.CreateNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// JoinNetwork entra em uma sala
func (s *SignalingClient) JoinNetwork(networkID string, pin string, computername string) (*signaling_models.JoinNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Joining network: %s", networkID)

	// Criar payload para join network
	payload := &signaling_models.JoinNetworkRequest{
		BaseRequest:  signaling_models.BaseRequest{},
		NetworkID:    networkID,
		PIN:          pin,
		ComputerName: computername,
	}

	// Enviar solicitação para entrar na sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(signaling_models.TypeJoinNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(signaling_models.JoinNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// ConnectNetwork conecta a uma sala previamente associada
func (s *SignalingClient) ConnectNetwork(networkID string, computerName string) (*signaling_models.ConnectNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Connecting to network: %s", networkID)

	// Criar payload para connect network
	payload := &signaling_models.ConnectNetworkRequest{
		BaseRequest:  signaling_models.BaseRequest{},
		NetworkID:    networkID,
		ComputerName: computerName,
	}

	// Enviar solicitação para conectar à sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(signaling_models.TypeConnectNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(signaling_models.ConnectNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// DisconnectNetwork desconecta de uma sala sem sair dela
func (s *SignalingClient) DisconnectNetwork(networkID string) (*signaling_models.DisconnectNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Disconnecting from network: %s", networkID)

	// Criar payload para disconnect network
	payload := &signaling_models.DisconnectNetworkRequest{
		BaseRequest: signaling_models.BaseRequest{},
		NetworkID:   networkID,
	}

	// Enviar solicitação para desconectar da sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(signaling_models.TypeDisconnectNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	// Check if we got a map response (like from TypeNetworkDisconnected)
	if respMap, ok := response.(map[string]interface{}); ok {
		// Extract network ID from the map
		networkID, _ := respMap["network_id"].(string)
		resp := signaling_models.DisconnectNetworkResponse{
			NetworkID: networkID,
		}
		return &resp, nil
	} else if resp, ok := response.(signaling_models.DisconnectNetworkResponse); ok {
		return &resp, nil
	}

	// For debugging the response type
	log.Printf("Unexpected response type: %T", response)
	return nil, errors.New("unexpected response type")
}

// LeaveNetwork sai de uma sala
func (s *SignalingClient) LeaveNetwork(networkID string) (*signaling_models.LeaveNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Leaving network: %s", networkID)

	// Criar payload para leave network
	payload := &signaling_models.LeaveNetworkRequest{
		BaseRequest: signaling_models.BaseRequest{},
		NetworkID:   networkID,
	}

	// Enviar solicitação para sair da sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(signaling_models.TypeLeaveNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(signaling_models.LeaveNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// RenameNetwork renomeia uma sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) RenameNetwork(networkID string, newName string) (*signaling_models.RenameResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Renaming network %s to %s", networkID, newName)

	// Criar payload para rename network
	payload := &signaling_models.RenameRequest{
		BaseRequest: signaling_models.BaseRequest{},
		NetworkID:   networkID,
		NetworkName: newName,
	}

	// Enviar solicitação para renomear a sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(signaling_models.TypeRename, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(signaling_models.RenameResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// KickComputer expulsa um usuário da sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) KickComputer(networkID string, targetID string) (*signaling_models.KickResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Kicking computer %s from network %s", targetID, networkID)

	// Criar payload para kick computer
	payload := &signaling_models.KickRequest{
		BaseRequest: signaling_models.BaseRequest{},
		NetworkID:   networkID,
		TargetID:    targetID,
	}

	// Enviar solicitação para expulsar o usuário usando a função de empacotamento
	response, err := s.sendPackagedMessage(signaling_models.TypeKick, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(signaling_models.KickResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// SendMessage envia uma mensagem para o servidor
func (s *SignalingClient) SendMessage(messageType signaling_models.MessageType, payload interface{}) (interface{}, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	// Enviar a mensagem usando a função de empacotamento
	return s.sendPackagedMessage(messageType, payload)
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
			// Check if the error is due to a closed connection
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) || websocket.IsUnexpectedCloseError(err) {
				log.Printf("WebSocket connection closed: %v", err)
			} else {
				log.Printf("Error reading message: %v", err)
			}
			s.Connected = false
			s.Conn = nil
			return
		}

		// Process the message here before passing to custom handler
		var sigMsg signaling_models.SignalingMessage
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
		case signaling_models.TypeError:
			{
				// Decode error message from base64
				var errorPayload map[string]string
				if err := json.Unmarshal(sigMsg.Payload, &errorPayload); err == nil {
					if errorMsg, ok := errorPayload["error"]; ok {
						log.Printf("Server error: %s", errorMsg)
						// Notify the handler about the error
						if s.MessageHandler != nil {
							s.MessageHandler(signaling_models.TypeError, sigMsg.Payload)
						}
					}
				} else {
					log.Printf("Failed to decode error payload: %v", err)
				}
			}

		case signaling_models.TypeNetworkDisconnected:
			{
				var response signaling_models.DisconnectNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal network disconnected response: %v", err)
				} else {
					log.Printf("Successfully disconnected from network: %s", response.NetworkID)
					// Notify the handler about the network disconnection
					if s.MessageHandler != nil {
						s.MessageHandler(signaling_models.TypeNetworkDisconnected, sigMsg.Payload)
					}
				}
			}

		case signaling_models.TypeNetworkJoined:
			{
				var response signaling_models.JoinNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal network joined response: %v", err)
				} else {
					log.Printf("Successfully joined network: %s (%s)", response.NetworkName, response.NetworkID)
					// Notify the handler about the network joined event
					if s.MessageHandler != nil {
						s.MessageHandler(signaling_models.TypeNetworkJoined, sigMsg.Payload)
					}
				}
			}

		case signaling_models.TypeNetworkCreated:
			{
				var response signaling_models.CreateNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal network created response: %v", err)
				} else {
					log.Printf("Successfully created network: %s (%s)", response.NetworkName, response.NetworkID)
					// Notify the handler about the network created event
					if s.MessageHandler != nil {
						s.MessageHandler(signaling_models.TypeNetworkCreated, sigMsg.Payload)
					}
				}
			}

		case signaling_models.TypeLeaveNetwork:
			{
				var response signaling_models.LeaveNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal leave network response: %v", err)
				} else {
					log.Printf("Successfully left network: %s", response.NetworkID)
					// Notify the handler about the network left event
					if s.MessageHandler != nil {
						s.MessageHandler(signaling_models.TypeLeaveNetwork, sigMsg.Payload)
					}
				}
			}

		case signaling_models.TypeKicked:
			{
				var notification signaling_models.KickedNotification
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
						s.MessageHandler(signaling_models.TypeKicked, sigMsg.Payload)
					}
				}
			}

		case signaling_models.TypeComputerJoined:
			{
				var notification signaling_models.ComputerJoinedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal computer joined notification: %v", err)
				} else {
					computerName := notification.ComputerName
					if computerName == "" {
						computerName = "Unknown computer"
					}
					log.Printf("New computer joined network %s: %s (Key: %s, IP: %s)", notification.NetworkID, computerName, notification.PublicKey, notification.ComputerIP)

					// Notify the handler about the computer joined event
					if s.MessageHandler != nil {
						s.MessageHandler(signaling_models.TypeComputerJoined, sigMsg.Payload)
					}
				}
			}

		case signaling_models.TypeComputerConnected:
			{
				var notification signaling_models.ComputerConnectedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal computer connected notification: %v", err)
				} else {
					computerName := notification.ComputerName
					if computerName == "" {
						computerName = "Unknown computer"
					}
					log.Printf("Computer connected to network %s: %s (Key: %s, IP: %s)", notification.NetworkID, computerName, notification.PublicKey, notification.ComputerIP)

					// Notify the handler about the computer connected event
					if s.MessageHandler != nil {
						s.MessageHandler(signaling_models.TypeComputerConnected, sigMsg.Payload)
					}
				}
			}

		case signaling_models.TypeComputerLeft:
			var notification signaling_models.ComputerLeftNotification
			if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
				log.Printf("Failed to unmarshal computer left notification: %v", err)
			} else {
				log.Printf("Computer left network %s: %s", notification.NetworkID, notification.PublicKey)

				// Notify the handler about the computer left event
				if s.MessageHandler != nil {
					s.MessageHandler(signaling_models.TypeComputerLeft, sigMsg.Payload)
				}
			}

		case signaling_models.TypeNetworkDeleted:
			{
				var notification signaling_models.NetworkDeletedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal network deleted notification: %v", err)
				} else {
					log.Printf("Network deleted: %s", notification.NetworkID)
					// Notify the handler about the network deleted event
					if s.MessageHandler != nil {
						s.MessageHandler(signaling_models.TypeNetworkDeleted, sigMsg.Payload)
					}
				}
			}

		case signaling_models.TypeComputerNetworks:
			{
				var response signaling_models.ComputerNetworksResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal computer networks response: %v", err)
				} else {
					log.Printf("=================== COMPUTER NETWORKS RECEIVED ===================")
					log.Printf("Total networks found: %d", len(response.Networks))
					log.Printf("==================================================================")

					for i, network := range response.Networks {
						log.Printf("Network %d: %s", i+1, network.NetworkName)
						log.Printf("  Network ID: %s", network.NetworkID)
						log.Printf("  Admin Public Key: %s", network.AdminPublicKey)
						log.Printf("  Joined at: %s", network.JoinedAt.Format(time.RFC1123))
						log.Printf("  Last connected: %s", network.LastConnected.Format(time.RFC1123))
						log.Printf("  Your Computer IP: %s", network.ComputerIP)

						log.Printf("  Connected computers (%d):", len(network.Computers))
						if len(network.Computers) > 0 {
							for j, computer := range network.Computers {
								log.Printf("    Computer %d: %s", j+1, computer.Name)
								log.Printf("      Public Key: %s", computer.PublicKey)
								log.Printf("      IP Address: %s", computer.ComputerIP)
								log.Printf("      Online: %t", computer.IsOnline)
							}
						} else {
							log.Printf("    No computers currently connected")
						}
						log.Printf("  ==================================================================")
					}

					// Notify the handler about the updated network list
					if s.MessageHandler != nil {
						s.MessageHandler(signaling_models.TypeComputerNetworks, sigMsg.Payload)
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
	_, err := s.sendPackagedMessage(signaling_models.TypePing, pingMessage)
	return err
}

// registerPendingRequest registers a message ID and returns a channel to receive the response
func (s *SignalingClient) registerPendingRequest(messageID string) chan signaling_models.SignalingMessage {
	s.pendingRequestsLock.Lock()
	defer s.pendingRequestsLock.Unlock()

	// Create channel for this request
	responseChan := make(chan signaling_models.SignalingMessage, 1)
	s.pendingRequests[messageID] = responseChan
	return responseChan
}

// handlePendingResponse routes responses to the appropriate waiting goroutine
func (s *SignalingClient) handlePendingResponse(msg signaling_models.SignalingMessage) bool {
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
func (s *SignalingClient) GetComputerNetworks() (*signaling_models.ComputerNetworksResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Requesting computer networks from server")

	// Create payload for the request
	payload := &signaling_models.GetComputerNetworksRequest{
		BaseRequest: signaling_models.BaseRequest{},
	}

	// Send request using the packaging function
	response, err := s.sendPackagedMessage(signaling_models.TypeGetComputerNetworks, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(signaling_models.ComputerNetworksResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}
