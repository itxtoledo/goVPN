package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/itxtoledo/govpn/libs/models"
	"github.com/itxtoledo/govpn/libs/network"
	"github.com/pion/webrtc/v3"
)

// Connection state constants
const (
	ConnectionStateDisconnected = iota
	ConnectionStateConnecting
	ConnectionStateConnected
)

// NetworkManager manages WebRTC connections
type NetworkManager struct {
	SignalServer          string                            // Endereço do servidor de sinalização
	IsConnected           bool                              // Indica se está conectado ao servidor de sinalização
	WSConn                *websocket.Conn                   // WebSocket connection com o servidor de sinalização
	RoomID                string                            // ID da sala atual
	RoomName              string                            // Nome da sala atual
	VirtualNetwork        *network.VirtualNetwork           // Rede virtual
	PublicKey             string                            // Chave pública serializada como base64
	PrivateKey            *rsa.PrivateKey                   // Chave privada
	PeersConn             map[string]*webrtc.PeerConnection // Conexões com peers por ID
	RTCConfig             webrtc.Configuration              // Configuração RTC
	DataChannels          map[string]*webrtc.DataChannel    // Canais de dados por ID
	PeersByPublicKey      map[string]bool                   // Peers por chave pública
	PeerUsernames         map[string]string                 // Usernames mapeados por chave pública
	VPNClient             *VPNClient                        // Referência para o cliente VPN
	wg                    sync.WaitGroup
	mu                    sync.RWMutex
	SignalServerConnected bool                            // Indica se está conectado ao servidor de sinalização
	Username              string                          // Nome de usuário do peer local
	MaxRetries            int                             // Maximum number of connection retry attempts
	CurrentRetry          int                             // Current retry attempt counter
	OnConnectionError     func(error)                     // Callback for connection errors
	OnRoomListUpdate      func([]models.Room)             // Callback for room list updates
	pendingRequests       map[string]chan models.Message  // Map to track requests by their ID
	messageCallbacks      map[string]func(models.Message) // Map for message callbacks by message ID
}

// NewNetworkManager cria um novo gerenciador de rede
func NewNetworkManager(vpn *VPNClient) *NetworkManager {
	// Definir servidor padrão com opção de ambiente
	signalServer := "ws://localhost:8080/ws"
	if serverEnv := vpn.GetConfig("SIGNAL_SERVER"); serverEnv != "" {
		signalServer = serverEnv
	}

	return &NetworkManager{
		VPNClient:    vpn,
		SignalServer: signalServer,
		PeersConn:    make(map[string]*webrtc.PeerConnection),
		RTCConfig: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
			},
		},
		IsConnected:           false,
		SignalServerConnected: false,
		PeersByPublicKey:      make(map[string]bool),
		MaxRetries:            3,
		CurrentRetry:          0,
		pendingRequests:       make(map[string]chan models.Message),
		messageCallbacks:      make(map[string]func(models.Message)),
	}
}

// Connect conecta ao servidor de sinalização
func (n *NetworkManager) Connect() error {
	u, err := url.Parse(n.SignalServer)
	if err != nil {
		n.SignalServerConnected = false
		return fmt.Errorf("invalid signaling server URL: %w", err)
	}

	log.Printf("Connecting to signaling server: %s", u.String())
	n.SignalServerConnected = false

	for n.CurrentRetry = 0; n.CurrentRetry < n.MaxRetries; n.CurrentRetry++ {
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("Connection attempt %d failed: %v", n.CurrentRetry+1, err)
			if n.OnConnectionError != nil {
				n.OnConnectionError(err)
			}
			continue
		}

		n.WSConn = conn
		n.IsConnected = true
		n.SignalServerConnected = true

		// Inicia a rotina para lidar com mensagens do servidor
		go n.handleWebSocketMessages()

		return nil
	}

	n.SignalServerConnected = false
	return fmt.Errorf("could not connect to signaling server at %s after %d attempts", n.SignalServer, n.MaxRetries)
}

// Disconnect desconecta do servidor de sinalização
func (n *NetworkManager) Disconnect() {
	if n.WSConn != nil {
		n.WSConn.Close()
		n.WSConn = nil
	}
	n.IsConnected = false
	n.SignalServerConnected = false
}

// GetRoomList obtém a lista de salas disponíveis apenas do banco de dados local,
// sem chamar o backend
func (n *NetworkManager) GetRoomList() error {
	// Carrega apenas as salas do banco de dados local
	localRooms, err := n.loadLocalRooms()
	if err != nil {
		log.Printf("Erro ao carregar salas locais: %v", err)
		return fmt.Errorf("erro ao carregar salas do banco de dados local: %w", err)
	}

	// Notifica a UI com as salas locais
	if n.OnRoomListUpdate != nil {
		n.OnRoomListUpdate(localRooms)
	}

	return nil
}

// loadLocalRooms carrega a lista de salas do banco de dados SQLite local
func (n *NetworkManager) loadLocalRooms() ([]models.Room, error) {
	var rooms []models.Room

	// Consulta as salas no banco de dados local
	rows, err := n.VPNClient.DB.Query(`
		SELECT id, name, password, last_connected 
		FROM rooms 
		ORDER BY last_connected DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar banco de dados: %w", err)
	}
	defer rows.Close()

	// Processa os resultados
	for rows.Next() {
		var room models.Room
		var lastConnected string

		err := rows.Scan(&room.ID, &room.Name, &room.Password, &lastConnected)
		if err != nil {
			log.Printf("Erro ao ler registro de sala: %v", err)
			continue
		}

		// Adiciona a sala à lista
		room.ClientCount = 0 // Não sabemos quantos clientes até conectar
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// JoinRoom entra em uma sala existente
func (n *NetworkManager) JoinRoom(roomID, password string) error {
	// Check if we're connected to the signaling server first
	if !n.SignalServerConnected {
		// Try to connect
		if err := n.Connect(); err != nil {
			// Display specific message about signaling server connection failure
			if n.VPNClient.UI != nil {
				n.VPNClient.UI.ShowMessage("Connection Error", "Could not connect to the signaling server. Please check your internet connection and try again.")
			}
			return fmt.Errorf("failed to connect to signaling server: %w", err)
		}
	}

	// Garante que as chaves RSA estão carregadas ou cria novas se necessário
	if err := n.VPNClient.loadOrGenerateKeys(); err != nil {
		return fmt.Errorf("erro ao obter chaves RSA: %w", err)
	}

	// Inicializa a rede virtual com o ID e senha da sala
	n.VirtualNetwork = network.NewVirtualNetwork(roomID, password)

	// Use default username if none is set
	username := n.Username
	if username == "" {
		username = "User" // Default username
	} else if len(username) > 10 {
		// Limita o nome de usuário a 10 caracteres
		username = username[:10]
	}

	msg := models.Message{
		Type:      "JoinRoom",
		RoomID:    roomID,
		Password:  password,
		PublicKey: n.VPNClient.getPublicKey(), // Usa apenas a chave pública para identificação
		Username:  username,                   // Include the username
	}

	// Envia a mensagem com timeout de 2 segundos (padrão)
	response, err := n.SendRequestWithID(msg, 0)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			if n.VPNClient.UI != nil {
				n.VPNClient.UI.ShowMessage("Error", "O servidor demorou muito para responder. Por favor, tente novamente mais tarde.")
			}
		}
		return err
	}

	// Verifica se a resposta é um erro
	if response.Type == "Error" {
		return fmt.Errorf("erro ao entrar na sala: %s", string(response.Data))
	}

	// Se a resposta for bem-sucedida, processa o resultado
	if response.Type == "RoomJoined" {
		n.VPNClient.CurrentRoom = response.RoomID
		n.RoomName = response.RoomName
		n.VPNClient.IsConnected = true
		log.Printf("Room successfully joined: %s (%s)", response.RoomID, response.RoomName)

		// Atualizar a interface de usuário para refletir a conexão
		if n.VPNClient.UI != nil && n.VPNClient.UI.HeaderComponent != nil {
			// Atualiza o IP, nome de usuário e informações da sala
			n.VPNClient.UI.HeaderComponent.updateIPInfo()
			n.VPNClient.UI.HeaderComponent.updateUsername()
			n.VPNClient.UI.HeaderComponent.updateRoomName()

			// Atualiza o botão de energia e a lista de rede
			n.VPNClient.UI.updatePowerButtonState()
			n.VPNClient.UI.refreshNetworkList()
		}

		return nil
	}

	// Se chegou aqui, a resposta não é nem erro nem sucesso
	return fmt.Errorf("resposta inesperada do servidor: %s", response.Type)
}

// CreateRoom cria uma nova sala
func (n *NetworkManager) CreateRoom(name, password string) error {
	// Check if we're connected to the signaling server first
	if !n.SignalServerConnected {
		// Try to connect
		if err := n.Connect(); err != nil {
			// Display specific message about signaling server connection failure
			if n.VPNClient.UI != nil {
				n.VPNClient.UI.ShowMessage("Connection Error", "Could not connect to the signaling server. Please check your internet connection and try again.")
			}
			return fmt.Errorf("failed to connect to signaling server: %w", err)
		}
	}

	// Garante que as chaves RSA estão carregadas ou cria novas se necessário
	if err := n.VPNClient.loadOrGenerateKeys(); err != nil {
		return fmt.Errorf("erro ao obter chaves RSA: %w", err)
	}

	// Verifica se a chave pública está disponível
	publicKey := n.VPNClient.getPublicKey()
	if publicKey == "" {
		return fmt.Errorf("não foi possível obter a chave pública")
	}

	msg := models.Message{
		Type:      "CreateRoom",
		RoomName:  name,
		Password:  password,
		PublicKey: publicKey,
	}

	// Envia a mensagem com timeout de 2 segundos (padrão)
	response, err := n.SendRequestWithID(msg, 0)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			if n.VPNClient.UI != nil {
				n.VPNClient.UI.ShowMessage("Error", "O servidor demorou muito para responder. Por favor, tente novamente mais tarde.")
			}
		}
		return err
	}

	// Verifica se a resposta é um erro
	if response.Type == "Error" {
		return fmt.Errorf("erro ao criar sala: %s", string(response.Data))
	}

	// Se a resposta for bem-sucedida, processa o resultado
	if response.Type == "RoomCreated" {
		n.VPNClient.CurrentRoom = response.RoomID
		n.RoomName = response.RoomName
		n.VPNClient.IsConnected = true
		log.Printf("Room successfully created: %s (%s)", response.RoomID, response.RoomName)

		// Salva a sala no banco de dados local
		if err := n.saveRoomToLocalDB(response.RoomID, response.RoomName, password); err != nil {
			log.Printf("Aviso: Não foi possível salvar a sala no banco de dados local: %v", err)
			// Continuamos mesmo com erro ao salvar no banco local
		}

		return nil
	}

	// Se chegou aqui, a resposta não é nem erro nem sucesso
	return fmt.Errorf("resposta inesperada do servidor: %s", response.Type)
}

// LeaveRoom sai da sala atual
func (n *NetworkManager) LeaveRoom() error {
	if !n.IsConnected || n.VPNClient.CurrentRoom == "" {
		return nil
	}

	msg := models.Message{
		Type:      "LeaveRoom",
		RoomID:    n.VPNClient.CurrentRoom,
		PublicKey: n.VPNClient.getPublicKey(),
	}

	// Limpa as informações da sala
	n.RoomName = ""
	n.VirtualNetwork = nil

	return n.sendMessage(msg)
}

// sendMessage envia uma mensagem para o servidor de sinalização
func (n *NetworkManager) sendMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err := n.WSConn.WriteMessage(websocket.TextMessage, data); err != nil {
		return err
	}

	return nil
}

// handleWebSocketMessages processa as mensagens recebidas do servidor de sinalização
func (n *NetworkManager) handleWebSocketMessages() {
	for {
		_, message, err := n.WSConn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			n.IsConnected = false
			n.SignalServerConnected = false
			return
		}

		var msg models.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error deserializing message: %v", err)
			continue
		}

		// Processa a mensagem com base no tipo
		switch msg.Type {
		case "RoomList":
			n.handleRoomListMessage(message)
		case "RoomCreated":
			n.handleRoomCreatedMessage(msg)
		case "RoomJoined":
			n.handleRoomJoinedMessage(msg)
		case "PeerJoined":
			n.handlePeerJoinedMessage(msg)
		case "PeerLeft":
			n.handlePeerLeftMessage(msg)
		case "Offer":
			n.handleOfferMessage(msg)
		case "Answer":
			n.handleAnswerMessage(msg)
		case "Candidate":
			n.handleICECandidateMessage(msg)
		case "Error":
			n.handleErrorMessage(msg)
		}

		// Processa IDs de mensagens para callbacks ou respostas
		if !n.processMessageID(msg) {
			log.Printf("Message ID not processed: %s", msg.MessageID)
		}
	}
}

// Métodos para tratamento de mensagens específicas
func (n *NetworkManager) handleRoomListMessage(message []byte) {
	var roomListMsg struct {
		Type  string        `json:"type"`
		Rooms []models.Room `json:"rooms"`
	}

	if err := json.Unmarshal(message, &roomListMsg); err != nil {
		log.Printf("Error deserializing room list: %v", err)
		return
	}

	// Notifica a UI sobre a atualização da lista de salas
	if n.OnRoomListUpdate != nil {
		n.OnRoomListUpdate(roomListMsg.Rooms)
	}
}

func (n *NetworkManager) handleRoomCreatedMessage(msg models.Message) {
	n.VPNClient.CurrentRoom = msg.RoomID
	n.RoomName = msg.RoomName
	n.VPNClient.IsConnected = true
	log.Printf("Room successfully created: %s (%s)", msg.RoomID, msg.RoomName)
}

func (n *NetworkManager) handleRoomJoinedMessage(msg models.Message) {
	n.VPNClient.CurrentRoom = msg.RoomID
	n.RoomName = msg.RoomName
	n.VPNClient.IsConnected = true
	log.Printf("Joined room: %s (%s)", msg.RoomID, msg.RoomName)

	// Salva a sala no banco de dados local
	if password, err := n.getPasswordForRoom(msg.RoomID); err == nil {
		if err := n.saveRoomToLocalDB(msg.RoomID, msg.RoomName, password); err != nil {
			log.Printf("Aviso: Não foi possível salvar a sala no banco de dados local: %v", err)
		}
	} else {
		log.Printf("Aviso: Não foi possível recuperar senha para salvar sala: %v", err)
	}

	// Atualizar a interface de usuário para refletir a conexão
	if n.VPNClient.UI != nil && n.VPNClient.UI.HeaderComponent != nil {
		// Atualiza o IP, nome de usuário e informações da sala
		n.VPNClient.UI.HeaderComponent.updateIPInfo()
		n.VPNClient.UI.HeaderComponent.updateUsername()
		n.VPNClient.UI.HeaderComponent.updateRoomName()

		// Atualiza o botão de energia e a lista de rede
		n.VPNClient.UI.updatePowerButtonState()
		n.VPNClient.UI.refreshNetworkList()
	}
}

func (n *NetworkManager) handlePeerJoinedMessage(msg models.Message) {
	peerPublicKey := msg.PublicKey
	username := msg.Username // Extract username from the message

	log.Printf("New peer joined the room: %s (%s)", peerPublicKey, username)

	// Se a chave pública está presente, podemos usá-la como identificador principal
	if peerPublicKey != "" {
		// Armazena o status do peer como conectado
		n.PeersByPublicKey[peerPublicKey] = true

		// Initialize the PeerUsernames map if it doesn't exist
		if n.PeerUsernames == nil {
			n.PeerUsernames = make(map[string]string)
		}

		// Store the username for this peer
		if username != "" {
			n.PeerUsernames[peerPublicKey] = username
		} else {
			// Use a default name if none provided
			n.PeerUsernames[peerPublicKey] = "User"
		}

		// truncamos a chave pública para exibição por ser muito longa
		displayID := n.GetFormattedPublicKey(peerPublicKey)
		log.Printf("Using public key as peer identifier: %s", displayID)
	}

	// Iniciar WebRTC com o peer
	// Na implementação real, estabelecer uma conexão WebRTC e configurar o datachannel
}

func (n *NetworkManager) handlePeerLeftMessage(msg models.Message) {
	peerPublicKey := msg.PublicKey
	log.Printf("Peer left the room: %s", peerPublicKey)
	// Remover conexão WebRTC com o peer
	if n.VirtualNetwork != nil {
		n.VirtualNetwork.RemovePeer(peerPublicKey)
	}

	// Remove o peer dos mapeamentos
	delete(n.PeersByPublicKey, peerPublicKey)
}

func (n *NetworkManager) handleOfferMessage(msg models.Message) {
	// Implementar lógica para lidar com ofertas WebRTC
	log.Printf("Offer received from peer: %s", msg.PublicKey)

	// Aqui você criaria um PeerConnection, configuraria os canais de dados
	// e responderia com uma Answer
}

func (n *NetworkManager) handleAnswerMessage(msg models.Message) {
	// Implementar lógica para lidar com respostas WebRTC
	log.Printf("Answer received from peer: %s", msg.PublicKey)

	// Aqui você aplicaria a answer ao PeerConnection correspondente
}

func (n *NetworkManager) handleICECandidateMessage(msg models.Message) {
	// Implementar lógica para lidar com candidatos ICE
	log.Printf("ICE candidate received from peer: %s", msg.PublicKey)

	// Aqui você adicionaria o candidato ICE ao PeerConnection correspondente
}

func (n *NetworkManager) handleErrorMessage(msg models.Message) {
	errorMsg := string(msg.Data)
	log.Printf("Error received from server: %s", errorMsg)

	// Verifica se o erro é sobre chave pública duplicada
	if errorStr := string(msg.Data); len(errorStr) > 0 {
		// Busca por mensagens específicas relacionadas à chave pública
		if publicKeyError := checkPublicKeyError(errorStr); publicKeyError != "" {
			// Se encontrar uma sala existente, notifica o usuário
			// Isso deve ser tratado pela UI, que pode exibir uma mensagem específica
			if n.VPNClient.UI != nil {
				n.VPNClient.UI.ShowMessage("Room already exists", publicKeyError)
			}
		}
	}
}

// checkPublicKeyError verifica se a mensagem de erro contém informações sobre chave pública duplicada
// e extrai o ID da sala existente, se disponível
func checkPublicKeyError(errorMsg string) string {
	// Busca por padrões específicos de erro relacionados à chave pública duplicada
	// Isso depende exatamente de como o servidor formata a mensagem de erro
	if len(errorMsg) > 2 && errorMsg[0] == '"' && errorMsg[len(errorMsg)-1] == '"' {
		// Remove as aspas da mensagem JSON
		errorMsg = errorMsg[1 : len(errorMsg)-1]
	}

	// Vários padrões possíveis de erro de chave duplicada que o servidor pode retornar
	if contains(errorMsg, "public key has already created room") {
		return errorMsg
	} else if contains(errorMsg, "This public key has already") {
		return errorMsg
	}

	return ""
}

// contains verifica se uma string contém outra
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// GetFormattedPublicKey retorna uma versão formatada (truncada) da chave pública para exibição
func (n *NetworkManager) GetFormattedPublicKey(publicKey string) string {
	if len(publicKey) <= 16 {
		return publicKey
	}

	// Trunca a chave para exibição: primeiros 8 caracteres + ... + últimos 8 caracteres
	return publicKey[:8] + "..." + publicKey[len(publicKey)-8:]
}

// IsBackendConnected returns whether the network manager is connected to the backend server
func (n *NetworkManager) IsBackendConnected() bool {
	return n.WSConn != nil && n.IsConnected
}

// GetConnectionState returns the current connection state
func (n *NetworkManager) GetConnectionState() int {
	if n.WSConn == nil {
		return ConnectionStateDisconnected
	}
	if n.IsConnected {
		return ConnectionStateConnected
	}
	return ConnectionStateConnecting
}

// IsSignalServerConnected returns whether the network manager is connected to the signaling server
func (n *NetworkManager) IsSignalServerConnected() bool {
	return n.SignalServerConnected
}

// GenerateRandomID gera um ID aleatório em formato hexadecimal com base no comprimento especificado
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

// SendRequestWithID envia uma mensagem para o servidor com um ID único e aguarda a resposta
// com o mesmo ID, retornando-a ou um erro se o timeout ocorrer
func (n *NetworkManager) SendRequestWithID(msg models.Message, timeoutSeconds int) (models.Message, error) {
	// Se não for especificado um timeout, usa 2 segundos como padrão
	if timeoutSeconds <= 0 {
		timeoutSeconds = 2
	}

	// Gera um ID aleatório de 10 caracteres usando a função do pacote models
	requestID, err := models.GenerateMessageID(10)
	if err != nil {
		return models.Message{}, fmt.Errorf("erro ao gerar ID da requisição: %w", err)
	}

	// Define o ID na mensagem
	msg.MessageID = requestID

	// Cria um canal para receber a resposta
	responseChan := make(chan models.Message, 1)

	// Registra o canal no mapa de solicitações pendentes
	n.mu.Lock()
	n.pendingRequests[requestID] = responseChan
	n.mu.Unlock()

	// Garante que o canal é removido ao final
	defer func() {
		n.mu.Lock()
		delete(n.pendingRequests, requestID)
		n.mu.Unlock()
	}()

	// Envia a mensagem
	err = n.sendMessage(msg)
	if err != nil {
		return models.Message{}, fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	// Configura um timeout
	timeout := time.After(time.Duration(timeoutSeconds) * time.Second)

	// Aguarda a resposta ou o timeout
	select {
	case response := <-responseChan:
		return response, nil
	case <-timeout:
		return models.Message{}, fmt.Errorf("timeout esperando resposta do servidor após %d segundos", timeoutSeconds)
	}
}

// RegisterMessageCallback registra um callback para ser chamado quando uma
// mensagem com um determinado ID for recebida
func (n *NetworkManager) RegisterMessageCallback(messageID string, callback func(models.Message)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messageCallbacks[messageID] = callback
}

// UnregisterMessageCallback remove um callback para um determinado ID de mensagem
func (n *NetworkManager) UnregisterMessageCallback(messageID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.messageCallbacks, messageID)
}

// processMessageID processa uma mensagem recebida que contenha um ID
// Essa função deve ser chamada do handler de mensagens WebSocket
func (n *NetworkManager) processMessageID(msg models.Message) bool {
	// Se a mensagem tem um ID, verifica se é uma resposta para uma solicitação pendente
	if msg.MessageID != "" {
		n.mu.Lock()
		defer n.mu.Unlock()

		// Verifica se existe um canal esperando esta resposta
		if responseChan, exists := n.pendingRequests[msg.MessageID]; exists {
			// Envia a mensagem para o canal
			responseChan <- msg
			return true
		}

		// Verifica se existe um callback registrado para este ID
		if callback, exists := n.messageCallbacks[msg.MessageID]; exists {
			// Executa o callback em uma goroutine separada para não bloquear o processamento
			go callback(msg)
			return true
		}
	}

	// Se chegou aqui, não é uma resposta para uma solicitação pendente nem para um callback
	return false
}

// saveRoomToLocalDB salva ou atualiza uma sala no banco de dados local
func (n *NetworkManager) saveRoomToLocalDB(roomID, roomName, password string) error {
	// Verifica se o ID da sala é válido
	if roomID == "" {
		return fmt.Errorf("ID da sala inválido")
	}

	// Salva ou atualiza a entrada no banco de dados
	_, err := n.VPNClient.DB.Exec(
		"INSERT OR REPLACE INTO rooms (id, name, password, last_connected) VALUES (?, ?, ?, CURRENT_TIMESTAMP)",
		roomID, roomName, password,
	)
	if err != nil {
		return fmt.Errorf("erro ao salvar sala no banco de dados: %w", err)
	}

	log.Printf("Sala %s (%s) salva no banco de dados local", roomID, roomName)
	return nil
}

// getPasswordForRoom obtém a senha para uma sala específica armazenada temporariamente
// durante o processo de conexão
func (n *NetworkManager) getPasswordForRoom(roomID string) (string, error) {

	// Se não conseguimos recuperar do VirtualNetwork, tentamos buscar do banco de dados
	var password string
	err := n.VPNClient.DB.QueryRow("SELECT password FROM rooms WHERE id = ?", roomID).Scan(&password)
	if err != nil {
		return "", fmt.Errorf("sala não encontrada no banco de dados: %w", err)
	}

	return password, nil
}
