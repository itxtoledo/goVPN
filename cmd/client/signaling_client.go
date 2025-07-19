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

// SignalingMessageHandler define o tipo de função para lidar com mensagens de sinalização recebidas.
type SignalingMessageHandler func(messageType models.MessageType, payload []byte)

// SignalingClient representa uma conexão com o servidor de sinalização
type SignalingClient struct {
	VPNClient      *VPNClient
	Conn           *websocket.Conn
	ServerAddress  string
	Connected      bool
	LastHeartbeat  time.Time
	MessageHandler SignalingMessageHandler // Usar o novo tipo de função
	PublicKeyStr   string                  // Public key string to identify this client

	// System to track pending requests by message ID
	pendingRequests     map[string]chan models.SignalingMessage
	pendingRequestsLock sync.Mutex
}

// NewSignalingClient cria uma nova instância do servidor de sinalização
func NewSignalingClient(publicKey string, handler SignalingMessageHandler) *SignalingClient {
	return &SignalingClient{
		Connected:       false,
		LastHeartbeat:   time.Now(),
		PublicKeyStr:    publicKey,
		MessageHandler:  handler, // Atribuir o handler passado
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
	headers["Computer-Agent"] = []string{"goVPN-Client/1.0"}

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
	case models.TypeCreateNetwork:
		if response.Type == models.TypeNetworkCreated {
			var resp models.CreateNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal create network response: %v", err)
			}
			return resp, nil
		}

	case models.TypeJoinNetwork:
		if response.Type == models.TypeNetworkJoined {
			var resp models.JoinNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal join network response: %v", err)
			}
			return resp, nil
		}

	case models.TypeConnectNetwork:
		if response.Type == models.TypeNetworkConnected {
			var resp models.ConnectNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal connect network response: %v", err)
			}
			return resp, nil
		}

	case models.TypeDisconnectNetwork:
		if response.Type == models.TypeDisconnectNetwork || response.Type == models.TypeNetworkDisconnected {
			var resp models.DisconnectNetworkResponse
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

	case models.TypeLeaveNetwork:
		if response.Type == models.TypeLeaveNetwork {
			var resp models.LeaveNetworkResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal leave network response: %v", err)
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

	case models.TypeGetComputerNetworks:
		if response.Type == models.TypeComputerNetworks {
			var resp models.ComputerNetworksResponse
			if err := json.Unmarshal(response.Payload, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal computer networks response: %v", err)
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

// CreateNetwork cria uma nova sala no servidor
func (s *SignalingClient) CreateNetwork(name string, password string) (*models.CreateNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Creating network: %s", name)

	// Criar payload para a requisição
	payload := &models.CreateNetworkRequest{
		BaseRequest: models.BaseRequest{},
		NetworkName: name,
		Password:    password,
	}

	// Enviar solicitação de criação de sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeCreateNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.CreateNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// JoinNetwork entra em uma sala
func (s *SignalingClient) JoinNetwork(networkID string, password string, computername string) (*models.JoinNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Joining network: %s", networkID)

	// Criar payload para join network
	payload := &models.JoinNetworkRequest{
		BaseRequest:  models.BaseRequest{},
		NetworkID:    networkID,
		Password:     password,
		ComputerName: computername,
	}

	// Enviar solicitação para entrar na sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeJoinNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.JoinNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// ConnectNetwork conecta a uma sala previamente associada
func (s *SignalingClient) ConnectNetwork(networkID string, computerName string) (*models.ConnectNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Connecting to network: %s", networkID)

	// Criar payload para connect network
	payload := &models.ConnectNetworkRequest{
		BaseRequest:  models.BaseRequest{},
		NetworkID:    networkID,
		ComputerName: computerName,
	}

	// Enviar solicitação para conectar à sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeConnectNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.ConnectNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// DisconnectNetwork desconecta de uma sala sem sair dela
func (s *SignalingClient) DisconnectNetwork(networkID string) (*models.DisconnectNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Disconnecting from network: %s", networkID)

	// Criar payload para disconnect network
	payload := &models.DisconnectNetworkRequest{
		BaseRequest: models.BaseRequest{},
		NetworkID:   networkID,
	}

	// Enviar solicitação para desconectar da sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeDisconnectNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	// Check if we got a map response (like from TypeNetworkDisconnected)
	if respMap, ok := response.(map[string]interface{}); ok {
		// Extract network ID from the map
		networkID, _ := respMap["network_id"].(string)
		resp := models.DisconnectNetworkResponse{
			NetworkID: networkID,
		}
		return &resp, nil
	} else if resp, ok := response.(models.DisconnectNetworkResponse); ok {
		return &resp, nil
	}

	// For debugging the response type
	log.Printf("Unexpected response type: %T", response)
	return nil, errors.New("unexpected response type")
}

// LeaveNetwork sai de uma sala
func (s *SignalingClient) LeaveNetwork(networkID string) (*models.LeaveNetworkResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Leaving network: %s", networkID)

	// Criar payload para leave network
	payload := &models.LeaveNetworkRequest{
		BaseRequest: models.BaseRequest{},
		NetworkID:   networkID,
	}

	// Enviar solicitação para sair da sala usando a função de empacotamento
	response, err := s.sendPackagedMessage(models.TypeLeaveNetwork, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.LeaveNetworkResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}

// RenameNetwork renomeia uma sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) RenameNetwork(networkID string, newName string) (*models.RenameResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Renaming network %s to %s", networkID, newName)

	// Criar payload para rename network
	payload := &models.RenameRequest{
		BaseRequest: models.BaseRequest{},
		NetworkID:   networkID,
		NetworkName: newName,
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

// KickComputer expulsa um usuário da sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) KickComputer(networkID string, targetID string) (*models.KickResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Kicking computer %s from network %s", targetID, networkID)

	// Criar payload para kick computer
	payload := &models.KickRequest{
		BaseRequest: models.BaseRequest{},
		NetworkID:   networkID,
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
			{
				// Decode error message from base64
				var errorPayload map[string]string
				if err := json.Unmarshal(sigMsg.Payload, &errorPayload); err == nil {
					if errorMsg, ok := errorPayload["error"]; ok {
						log.Printf("Server error: %s", errorMsg)
						// Notify the handler about the error
						if s.MessageHandler != nil {
							s.MessageHandler(models.TypeError, sigMsg.Payload)
						}
					}
				} else {
					log.Printf("Failed to decode error payload: %v", err)
				}
			}

		case models.TypeNetworkDisconnected:
			{
				var response models.DisconnectNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal network disconnected response: %v", err)
				} else {
					log.Printf("Successfully disconnected from network: %s", response.NetworkID)
					// Handle client disconnecting from network
					if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
						// Clean up computers list when disconnecting from a network
						s.VPNClient.NetworkManager.Computers = []Computer{}
						// Notify the handler about the network disconnection
						if s.MessageHandler != nil {
							s.MessageHandler(models.TypeNetworkDisconnected, sigMsg.Payload)
						}
					}
				}
			}

		case models.TypeNetworkJoined:
			{
				var response models.JoinNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal network joined response: %v", err)
				} else {
					log.Printf("Successfully joined network: %s (%s)", response.NetworkName, response.NetworkID)
					// Notify the handler about the network joined event
					if s.MessageHandler != nil {
						s.MessageHandler(models.TypeNetworkJoined, sigMsg.Payload)
					}

					// Initialize computers list for the network
					if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
						// Initialize with just this computer for now
						s.VPNClient.NetworkManager.Computers = []Computer{
							{
								ID:       s.VPNClient.PublicKeyStr,
								Name:     s.VPNClient.ComputerName,
								OwnerID:  s.VPNClient.PublicKeyStr,
								IsOnline: true,
							},
						}
					}
				}
			}

		case models.TypeNetworkCreated:
			{
				var response models.CreateNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal network created response: %v", err)
				} else {
					log.Printf("Successfully created network: %s (%s)", response.NetworkName, response.NetworkID)
					// Notify the handler about the network created event
					if s.MessageHandler != nil {
						s.MessageHandler(models.TypeNetworkCreated, sigMsg.Payload)
					}

					// Initialize computers list for the network
					if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
						// When creating a network, just add this computer as the only member
						s.VPNClient.NetworkManager.Computers = []Computer{
							{
								ID:       s.VPNClient.PublicKeyStr,
								Name:     s.VPNClient.ComputerName,
								OwnerID:  s.VPNClient.PublicKeyStr,
								IsOnline: true,
							},
						}
					}
				}
			}

		case models.TypeLeaveNetwork:
			{
				var response models.LeaveNetworkResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal leave network response: %v", err)
				} else {
					log.Printf("Successfully left network: %s", response.NetworkID)
					// Notify the handler about the network left event
					if s.MessageHandler != nil {
						s.MessageHandler(models.TypeLeaveNetwork, sigMsg.Payload)
					}
				}
			}

		case models.TypeKicked:
			{
				var notification models.KickedNotification
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
						s.MessageHandler(models.TypeKicked, sigMsg.Payload)
					}
				}
			}

		case models.TypeComputerJoined:
			{
				var notification models.ComputerJoinedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal computer joined notification: %v", err)
				} else {
					computerName := notification.ComputerName
					if computerName == "" {
						computerName = "Unknown computer"
					}
					log.Printf("New computer joined network %s: %s (Key: %s, IP: %s)", notification.NetworkID, computerName, notification.PublicKey, notification.PeerIP)

					// Add the new computer to the computers list
					if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
						// Skip if this is our own public key
						if notification.PublicKey != s.VPNClient.PublicKeyStr {
							// Add the new computer to the computers list
							newComputer := Computer{
								ID:       notification.PublicKey,
								Name:     computerName,
								OwnerID:  notification.PublicKey,
								IsOnline: true,
								PeerIP:   notification.PeerIP,
							}

							s.VPNClient.NetworkManager.Computers = append(
								s.VPNClient.NetworkManager.Computers,
								newComputer,
							)

							// Notify the handler about the computer joined event
							if s.MessageHandler != nil {
								s.MessageHandler(models.TypeComputerJoined, sigMsg.Payload)
							}
						}
					}
				}
			}

		case models.TypeComputerConnected:
			{
				var notification models.ComputerConnectedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal computer connected notification: %v", err)
				} else {
					computerName := notification.ComputerName
					if computerName == "" {
						computerName = "Unknown computer"
					}
					log.Printf("Computer connected to network %s: %s (Key: %s, IP: %s)", notification.NetworkID, computerName, notification.PublicKey, notification.PeerIP)

					if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
						// Update the status of the connected computer
						found := false
						for i, comp := range s.VPNClient.NetworkManager.Computers {
							if comp.ID == notification.PublicKey {
								s.VPNClient.NetworkManager.Computers[i].IsOnline = true
								s.VPNClient.NetworkManager.Computers[i].PeerIP = notification.PeerIP
								found = true
								break
							}
						}
						if !found {
							// If for some reason the computer wasn't in the list (e.g., joined while we were offline), add it
							newComputer := Computer{
								ID:       notification.PublicKey,
								Name:     computerName,
								OwnerID:  notification.PublicKey,
								IsOnline: true,
								PeerIP:   notification.PeerIP,
							}
							s.VPNClient.NetworkManager.Computers = append(s.VPNClient.NetworkManager.Computers, newComputer)
						}

						// Notify the handler about the computer connected event
						if s.MessageHandler != nil {
							s.MessageHandler(models.TypeComputerConnected, sigMsg.Payload)
						}
					}
				}
			}

		case models.TypeComputerLeft:
			var notification models.ComputerLeftNotification
			if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
				log.Printf("Failed to unmarshal computer left notification: %v", err)
			} else {
				log.Printf("Computer left network %s: %s", notification.NetworkID, notification.PublicKey)

				// Remove the computer from the computers list
				if s.VPNClient != nil && s.VPNClient.NetworkManager != nil && len(s.VPNClient.NetworkManager.Computers) > 0 {
					// Skip if this is our own public key
					if notification.PublicKey != s.VPNClient.PublicKeyStr {
						// Find and remove the computer from the computers list
						updatedComputers := []Computer{}
						for _, computer := range s.VPNClient.NetworkManager.Computers {
							if computer.ID != notification.PublicKey {
								updatedComputers = append(updatedComputers, computer)
							}
						}
						s.VPNClient.NetworkManager.Computers = updatedComputers

						// Notify the handler about the computer left event
						if s.MessageHandler != nil {
							s.MessageHandler(models.TypeComputerLeft, sigMsg.Payload)
						}
					}
				}
			}

		case models.TypeNetworkDeleted:
			{
				var notification models.NetworkDeletedNotification
				if err := json.Unmarshal(sigMsg.Payload, &notification); err != nil {
					log.Printf("Failed to unmarshal network deleted notification: %v", err)
				} else {
					log.Printf("Network deleted: %s", notification.NetworkID)
					// Clear computers list when a network is deleted
					if s.VPNClient != nil && s.VPNClient.NetworkManager != nil {
						s.VPNClient.NetworkManager.Computers = []Computer{}
					}
					s.VPNClient.NetworkManager.HandleNetworkDeleted(notification.NetworkID)
				}
			}

		case models.TypeComputerNetworks:
			{
				var response models.ComputerNetworksResponse
				if err := json.Unmarshal(sigMsg.Payload, &response); err != nil {
					log.Printf("Failed to unmarshal computer networks response: %v", err)
				} else {
					log.Printf("========== COMPUTER NETWORKS RECEIVED ==========")
					log.Printf("Total networks found: %d", len(response.Networks))

					// Create a slice to store the converted networks

					for i, network := range response.Networks {
						log.Printf("Network %d: %s (ID: %s)", i+1, network.NetworkName, network.NetworkID)
						log.Printf("  Joined at: %s", network.JoinedAt.Format(time.RFC1123))
						log.Printf("  Last connected: %s", network.LastConnected.Format(time.RFC1123))
						log.Printf("  ---")
					}

					log.Printf("========================================")

					// Notify the handler about the updated network list
					if s.MessageHandler != nil {
						s.MessageHandler(models.TypeComputerNetworks, sigMsg.Payload)
					}
				}
			}

		case models.TypeClientIPInfo:
			{
				var ipInfo models.ClientIPInfoResponse
				if err := json.Unmarshal(sigMsg.Payload, &ipInfo); err != nil {
					log.Printf("Failed to unmarshal client IP info: %v", err)
				} else {
					log.Printf("Received client IP info - IPv4: %s, IPv6: %s", ipInfo.IPv4, ipInfo.IPv6)

					// Notify the handler about the IP info
					if s.MessageHandler != nil {
						s.MessageHandler(models.TypeClientIPInfo, sigMsg.Payload)
					}
				}
			}
		}

		// Also pass to custom handler if configured
		// Note: The MessageHandler is already called for specific message types above
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

// GetComputerNetworks requests all networks the computer has joined from the server
func (s *SignalingClient) GetComputerNetworks() (*models.ComputerNetworksResponse, error) {
	if !s.Connected || s.Conn == nil {
		return nil, errors.New("not connected to server")
	}

	log.Printf("Requesting computer networks from server")

	// Create payload for the request
	payload := &models.GetComputerNetworksRequest{
		BaseRequest: models.BaseRequest{},
	}

	// Send request using the packaging function
	response, err := s.sendPackagedMessage(models.TypeGetComputerNetworks, payload)
	if err != nil {
		return nil, err
	}

	// Convert the response to the expected type
	if resp, ok := response.(models.ComputerNetworksResponse); ok {
		return &resp, nil
	}

	return nil, errors.New("unexpected response type")
}
