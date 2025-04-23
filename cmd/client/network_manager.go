package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

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

// NetworkManager gerencia as conexões de rede do cliente
type NetworkManager struct {
	VPN               *VPNClient
	WSConn            *websocket.Conn
	PeerConnections   map[string]*webrtc.PeerConnection // Mapeia PublicKey -> PeerConnection
	SignalServer      string
	ICEServers        []webrtc.ICEServer
	VirtualNetwork    *network.VirtualNetwork
	IsConnected       bool
	RoomName          string
	OnRoomListUpdate  func([]models.Room)
	OnConnectionError func(error) // Callback para erros de conexão
	MaxRetries        int         // Número máximo de tentativas de conexão
	CurrentRetry      int         // Contagem atual de tentativas

	// Mapeamento de peers pela chave pública
	PeersByPublicKey map[string]bool // Mapeia PublicKey -> status conectado
}

// NewNetworkManager cria um novo gerenciador de rede
func NewNetworkManager(vpn *VPNClient) *NetworkManager {
	// Definir servidor padrão com opção de ambiente
	signalServer := "ws://localhost:8080/ws"
	if serverEnv := vpn.GetConfig("SIGNAL_SERVER"); serverEnv != "" {
		signalServer = serverEnv
	}

	return &NetworkManager{
		VPN:             vpn,
		SignalServer:    signalServer,
		PeerConnections: make(map[string]*webrtc.PeerConnection),
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
		IsConnected:      false,
		PeersByPublicKey: make(map[string]bool),
		MaxRetries:       3,
		CurrentRetry:     0,
	}
}

// Connect conecta ao servidor de sinalização
func (n *NetworkManager) Connect() error {
	u, err := url.Parse(n.SignalServer)
	if err != nil {
		return err
	}

	log.Printf("Conectando ao servidor de sinalização: %s", u.String())

	for n.CurrentRetry = 0; n.CurrentRetry < n.MaxRetries; n.CurrentRetry++ {
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("Tentativa %d de conexão falhou: %v", n.CurrentRetry+1, err)
			if n.OnConnectionError != nil {
				n.OnConnectionError(err)
			}
			continue
		}

		n.WSConn = conn
		n.IsConnected = true

		// Inicia a rotina para lidar com mensagens do servidor
		go n.handleWebSocketMessages()

		return nil
	}

	return fmt.Errorf("não foi possível conectar ao servidor de sinalização após %d tentativas", n.MaxRetries)
}

// Disconnect desconecta do servidor de sinalização
func (n *NetworkManager) Disconnect() {
	if n.WSConn != nil {
		n.WSConn.Close()
		n.WSConn = nil
	}
	n.IsConnected = false
}

// GetRoomList obtém a lista de salas disponíveis
func (n *NetworkManager) GetRoomList() error {
	if !n.IsConnected {
		if err := n.Connect(); err != nil {
			return err
		}
	}

	msg := struct {
		Type string `json:"type"`
	}{
		Type: "getRoomList",
	}

	return n.sendMessage(msg)
}

// JoinRoom entra em uma sala existente
func (n *NetworkManager) JoinRoom(roomID, password string) error {
	if !n.IsConnected {
		if err := n.Connect(); err != nil {
			return err
		}
	}

	// Garante que as chaves RSA estão carregadas ou cria novas se necessário
	if err := n.VPN.loadOrGenerateKeys(); err != nil {
		return fmt.Errorf("erro ao obter chaves RSA: %w", err)
	}

	// Inicializa a rede virtual com o ID e senha da sala
	n.VirtualNetwork = network.NewVirtualNetwork(roomID, password)

	msg := models.Message{
		Type:      "JoinRoom",
		RoomID:    roomID,
		Password:  password,
		PublicKey: n.VPN.getPublicKey(), // Usa apenas a chave pública para identificação
	}

	return n.sendMessage(msg)
}

// CreateRoom cria uma nova sala
func (n *NetworkManager) CreateRoom(name, password string) error {
	if !n.IsConnected {
		if err := n.Connect(); err != nil {
			return err
		}
	}

	// Garante que as chaves RSA estão carregadas ou cria novas se necessário
	if err := n.VPN.loadOrGenerateKeys(); err != nil {
		return fmt.Errorf("erro ao obter chaves RSA: %w", err)
	}

	// Verifica se a chave pública está disponível
	publicKey := n.VPN.getPublicKey()
	if publicKey == "" {
		return fmt.Errorf("não foi possível obter a chave pública")
	}

	msg := models.Message{
		Type:      "CreateRoom",
		RoomName:  name,
		Password:  password,
		PublicKey: publicKey,
	}

	return n.sendMessage(msg)
}

// LeaveRoom sai da sala atual
func (n *NetworkManager) LeaveRoom() error {
	if !n.IsConnected || n.VPN.CurrentRoom == "" {
		return nil
	}

	msg := models.Message{
		Type:      "LeaveRoom",
		RoomID:    n.VPN.CurrentRoom,
		PublicKey: n.VPN.getPublicKey(),
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
			log.Printf("Erro ao ler mensagem: %v", err)
			n.IsConnected = false
			return
		}

		var msg models.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Erro ao desserializar mensagem: %v", err)
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
	}
}

// Métodos para tratamento de mensagens específicas
func (n *NetworkManager) handleRoomListMessage(message []byte) {
	var roomListMsg struct {
		Type  string        `json:"type"`
		Rooms []models.Room `json:"rooms"`
	}

	if err := json.Unmarshal(message, &roomListMsg); err != nil {
		log.Printf("Erro ao desserializar lista de salas: %v", err)
		return
	}

	// Notifica a UI sobre a atualização da lista de salas
	if n.OnRoomListUpdate != nil {
		n.OnRoomListUpdate(roomListMsg.Rooms)
	}
}

func (n *NetworkManager) handleRoomCreatedMessage(msg models.Message) {
	n.VPN.CurrentRoom = msg.RoomID
	n.RoomName = msg.RoomName
	n.VPN.IsConnected = true
	log.Printf("Sala criada com sucesso: %s (%s)", msg.RoomID, msg.RoomName)
}

func (n *NetworkManager) handleRoomJoinedMessage(msg models.Message) {
	n.VPN.CurrentRoom = msg.RoomID
	n.RoomName = msg.RoomName
	log.Printf("Entrou na sala: %s (%s)", msg.RoomID, msg.RoomName)
}

func (n *NetworkManager) handlePeerJoinedMessage(msg models.Message) {
	peerPublicKey := msg.PublicKey // Agora também recebemos a chave pública do peer

	log.Printf("Novo peer entrou na sala: %s", peerPublicKey)

	// Se a chave pública está presente, podemos usá-la como identificador principal
	if peerPublicKey != "" {
		// Armazena o status do peer como conectado
		n.PeersByPublicKey[peerPublicKey] = true

		// truncamos a chave pública para exibição por ser muito longa
		displayID := n.GetFormattedPublicKey(peerPublicKey)
		log.Printf("Usando chave pública como identificador do peer: %s", displayID)

		// Aqui você poderia inicializar as estruturas necessárias usando
		// a chave pública como identificador
	}

	// Iniciar WebRTC com o peer
	// Na implementação real, estabelecer uma conexão WebRTC e configurar o datachannel
}

func (n *NetworkManager) handlePeerLeftMessage(msg models.Message) {
	peerPublicKey := msg.PublicKey
	log.Printf("Peer saiu da sala: %s", peerPublicKey)
	// Remover conexão WebRTC com o peer
	if n.VirtualNetwork != nil {
		n.VirtualNetwork.RemovePeer(peerPublicKey)
	}

	// Remove o peer dos mapeamentos
	delete(n.PeersByPublicKey, peerPublicKey)
}

func (n *NetworkManager) handleOfferMessage(msg models.Message) {
	// Implementar lógica para lidar com ofertas WebRTC
	log.Printf("Oferta recebida do peer: %s", msg.PublicKey)

	// Aqui você criaria um PeerConnection, configuraria os canais de dados
	// e responderia com uma Answer
}

func (n *NetworkManager) handleAnswerMessage(msg models.Message) {
	// Implementar lógica para lidar com respostas WebRTC
	log.Printf("Resposta recebida do peer: %s", msg.PublicKey)

	// Aqui você aplicaria a answer ao PeerConnection correspondente
}

func (n *NetworkManager) handleICECandidateMessage(msg models.Message) {
	// Implementar lógica para lidar com candidatos ICE
	log.Printf("Candidato ICE recebido do peer: %s", msg.PublicKey)

	// Aqui você adicionaria o candidato ICE ao PeerConnection correspondente
}

func (n *NetworkManager) handleErrorMessage(msg models.Message) {
	errorMsg := string(msg.Data)
	log.Printf("Erro recebido do servidor: %s", errorMsg)

	// Verifica se o erro é sobre chave pública duplicada
	if errorStr := string(msg.Data); len(errorStr) > 0 {
		// Busca por mensagens específicas relacionadas à chave pública
		if publicKeyError := checkPublicKeyError(errorStr); publicKeyError != "" {
			// Se encontrar uma sala existente, notifica o usuário
			// Isso deve ser tratado pela UI, que pode exibir uma mensagem específica
			if n.VPN.UI != nil {
				n.VPN.UI.ShowMessage("Sala já existe", publicKeyError)
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
