// filepath: /Computers/gustavotoledodesouza/Projects/fun/goVPN/cmd/server/websocket_server.go
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/itxtoledo/govpn/cmd/server/logger"
	"github.com/itxtoledo/govpn/libs/crypto_utils"
	"github.com/itxtoledo/govpn/libs/models"
)

// ServerNetwork extends the basic Network model with server-specific fields
type ServerNetwork struct {
	models.Network                   // Embed the Network from models package
	PublicKey      ed25519.PublicKey `json:"-"`          // Not stored in Supabase directly
	PublicKeyB64   string            `json:"public_key"` // Stored as base64 string in Supabase
	CreatedAt      time.Time         `json:"created_at"`
	LastActive     time.Time         `json:"last_active"`
}

// SupabaseNetwork is a struct for network data stored in Supabase
type SupabaseNetwork struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	PIN        string    `json:"pin"`
	PublicKey  string    `json:"public_key"` // Base64 encoded public key
	CreatedAt  time.Time `json:"created_at"`
	LastActive time.Time `json:"last_active"`
}

// WebSocketServer manages the WebSocket connections and network handling
type WebSocketServer struct {
	clients           map[*websocket.Conn]string   // Maps connection to networkID
	networks          map[string][]*websocket.Conn // Maps networkID to list of connections
	clientToPublicKey map[*websocket.Conn]string   // Maps connection to public key
	connectedPeers    map[string]map[string]bool   // Maps networkID to map of publicKey to connected status
	mu                sync.RWMutex
	config            Config
	supabaseManager   *SupabaseManager
	upgrader          websocket.Upgrader
	pinRegex          *regexp.Regexp

	// Server statistics
	statsManager *StatsManager

	// Graceful shutdown
	shutdownChan chan struct{}
	httpServer   *http.Server
	isShutdown   bool
}

func NewWebSocketServer(cfg Config) (*WebSocketServer, error) {
	pinRegex, err := models.PINRegex()
	if err != nil {
		return nil, fmt.Errorf("failed to compile pin pattern: %w", err)
	}

	supaMgr, err := NewSupabaseManager(cfg.SupabaseURL, cfg.SupabaseKey, cfg.SupabaseNetworksTable, cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create Supabase manager: %w", err)
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return cfg.AllowAllOrigins
		},
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
	}

	statsManager := NewStatsManager(cfg)

	return &WebSocketServer{
		clients:           make(map[*websocket.Conn]string),
		networks:          make(map[string][]*websocket.Conn),
		clientToPublicKey: make(map[*websocket.Conn]string),
		connectedPeers:    make(map[string]map[string]bool),
		config:            cfg,
		supabaseManager:   supaMgr,
		upgrader:          upgrader,
		pinRegex:          pinRegex,
		statsManager:      statsManager,
		shutdownChan:      make(chan struct{}),
		httpServer:        &http.Server{},
		isShutdown:        false,
	}, nil
}

func (s *WebSocketServer) generateUniqueIP(networkID string) (string, error) {
	usedIPs, err := s.supabaseManager.GetUsedIPsForNetwork(networkID)
	if err != nil {
		return "", fmt.Errorf("failed to get used IPs for network %s: %w", networkID, err)
	}

	usedIPSet := make(map[string]bool)
	for _, ip := range usedIPs {
		usedIPSet[ip] = true
	}

	const maxAttempts = 254 // Limit attempts to find an IP
	for i := 0; i < maxAttempts; i++ {
		ip := fmt.Sprintf("10.10.0.%d", i+1)
		if !usedIPSet[ip] {
			return ip, nil
		}
	}

	return "", fmt.Errorf("no available IPs in network %s", networkID)
}

func (s *WebSocketServer) HandleWebSocketEndpoint(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection", "error", err)
		return
	}
	defer conn.Close()

	s.statsManager.IncrementConnectionsTotal()
	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	publicKeyHeader := r.Header.Get("X-Client-ID")
	if publicKeyHeader != "" && s.config.LogLevel == "debug" {
		logger.Debug("Client connected with public key", "publicKey", publicKeyHeader)

		msgID, _ := models.GenerateMessageID()

		req := models.GetComputerNetworksRequest{
			BaseRequest: models.BaseRequest{
				PublicKey: publicKeyHeader,
			},
		}

		go s.handleGetComputerNetworksWithIP(conn, req, msgID, r)
	}

	for {
		var sigMsg models.SignalingMessage
		err := conn.ReadJSON(&sigMsg)
		if err != nil {
			s.handleDisconnect(conn)
			return
		}

		s.statsManager.IncrementMessagesProcessed()

		originalID := sigMsg.ID

		switch sigMsg.Type {
		case models.TypeCreateNetwork:
			var req models.CreateNetworkRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid create network request format", originalID)
				continue
			}

			s.handleCreateNetwork(conn, req, originalID)

		case models.TypeJoinNetwork:
			var req models.JoinNetworkRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid join network request format", originalID)
				continue
			}

			s.handleJoinNetwork(conn, req, originalID)

		case models.TypeConnectNetwork:
			var req models.ConnectNetworkRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid connect network request format", originalID)
				continue
			}

			s.handleConnectNetwork(conn, req, originalID)

		case models.TypeDisconnectNetwork:
			var req models.DisconnectNetworkRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid disconnect network request format", originalID)
				continue
			}

			s.handleDisconnectNetwork(conn, req, originalID)

		case models.TypeLeaveNetwork:
			var req models.LeaveNetworkRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid leave network request format", originalID)
				continue
			}

			s.handleLeaveNetwork(conn, req, originalID)

		case models.TypeKick:
			var req models.KickRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid kick request format", originalID)
				continue
			}

			s.handleKick(conn, req, originalID)

		case models.TypeRename:
			var req models.RenameRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid rename request format", originalID)
				continue
			}

			s.handleRename(conn, req, originalID)

		case models.TypePing:
			s.handlePing(conn, sigMsg.Payload, originalID)

		case models.TypeGetComputerNetworks:
			var req models.GetComputerNetworksRequest
			if err := json.Unmarshal(sigMsg.Payload, &req); err != nil {
				s.sendErrorSignal(conn, "Invalid get computer networks request format", originalID)
				continue
			}

			s.handleGetComputerNetworks(conn, req, originalID)

		default:
			logger.Warn("Unknown message type", "type", sigMsg.Type)
			if originalID != "" {
				s.sendErrorSignal(conn, "Unknown message type", originalID)
			}
		}
	}
}

func (s *WebSocketServer) sendErrorSignal(conn *websocket.Conn, errorMsg string, originalID string) {
	errPayload, _ := json.Marshal(map[string]string{"error": errorMsg})

	conn.WriteJSON(models.SignalingMessage{
		ID:      originalID,
		Type:    models.TypeError,
		Payload: errPayload,
	})
}

func (s *WebSocketServer) sendSignal(conn *websocket.Conn, msgType models.MessageType, payload interface{}, originalID string) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return conn.WriteJSON(models.SignalingMessage{
		ID:      originalID,
		Type:    msgType,
		Payload: payloadBytes,
	})
}

func (s *WebSocketServer) handleCreateNetwork(conn *websocket.Conn, req models.CreateNetworkRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.NetworkName == "" || req.PIN == "" || req.PublicKey == "" {
		s.sendErrorSignal(conn, "Network name, pin, and public key are required", originalID)
		return
	}

	if !s.pinRegex.MatchString(req.PIN) {
		s.sendErrorSignal(conn, "PIN does not match required pattern", originalID)
		return
	}

	hasNetwork, existingNetworkID, err := s.supabaseManager.PublicKeyHasNetwork(req.PublicKey)
	if err != nil {
		logger.Error("Error checking if public key has a network", "error", err)
	} else if hasNetwork {
		s.sendErrorSignal(conn, fmt.Sprintf("This public key has already created network: %s", existingNetworkID), originalID)
		return
	}

	networkID := models.GenerateNetworkID()

	exists, err := s.supabaseManager.NetworkExists(networkID)
	if err != nil {
		logger.Error("Error checking if network exists", "error", err)
	} else if exists {
		s.sendErrorSignal(conn, "Network ID conflict, please try again", originalID)
		return
	}

	pubKey, err := crypto_utils.ParsePublicKey(req.PublicKey)
	if err != nil {
		s.sendErrorSignal(conn, "Invalid public key format", originalID)
		return
	}

	network := ServerNetwork{
		Network: models.Network{
			ID:   networkID,
			Name: req.NetworkName,
			PIN:  req.PIN,
		},
		PublicKey:    pubKey,
		PublicKeyB64: req.PublicKey,
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
	}

	err = s.supabaseManager.CreateNetwork(network)
	if err != nil {
		logger.Error("Error creating network in Supabase", "error", err)
		s.sendErrorSignal(conn, "Error creating network in database", originalID)
		return
	}

	creatorIP := "10.10.0.1"
	err = s.supabaseManager.AddComputerToNetwork(networkID, req.PublicKey, "Owner", creatorIP)
	if err != nil {
		logger.Error("Error adding network owner to computer_networks", "error", err)
	}

	s.clientToPublicKey[conn] = req.PublicKey

	s.clients[conn] = networkID

	if _, exists := s.networks[networkID]; !exists {
		s.networks[networkID] = []*websocket.Conn{}
	}
	s.networks[networkID] = append(s.networks[networkID], conn)

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		logger.Info("Network created",
			"networkID", networkID,
			"networkName", req.NetworkName,
			"clientAddr", conn.RemoteAddr().String())
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	responsePayload := map[string]interface{}{
		"network_id":   networkID,
		"network_name": req.NetworkName,
		"pin":          req.PIN,
		"public_key":   req.PublicKey,
		"peer_ip":      creatorIP,
	}

	s.sendSignal(conn, models.TypeNetworkCreated, responsePayload, originalID)
}

func (s *WebSocketServer) handleJoinNetwork(conn *websocket.Conn, req models.JoinNetworkRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	network, err := s.supabaseManager.GetNetwork(req.NetworkID)
	if err != nil {
		s.sendErrorSignal(conn, "Network does not exist", originalID)
		return
	}

	if req.PIN != network.PIN {
		s.sendErrorSignal(conn, "Incorrect PIN", originalID)
		return
	}

	if req.PublicKey == "" {
		s.sendErrorSignal(conn, "Public key is required", originalID)
		return
	}

	connections := s.networks[req.NetworkID]
	if len(connections) >= s.config.MaxClientsPerNetwork {
		s.sendErrorSignal(conn, "Network is full", originalID)
		return
	}

	var assignedIP string
	isInNetwork, err := s.supabaseManager.IsComputerInNetwork(req.NetworkID, req.PublicKey)
	if err != nil {
		logger.Error("Error checking if computer is in network", "error", err)
		s.sendErrorSignal(conn, "Error checking network membership", originalID)
		return
	}

	if !isInNetwork {
		// Assign a new IP if not already in network
		ip, err := s.generateUniqueIP(req.NetworkID)
		if err != nil {
			s.sendErrorSignal(conn, "Failed to assign IP address", originalID)
			return
		}
		assignedIP = ip

		err = s.supabaseManager.AddComputerToNetwork(req.NetworkID, req.PublicKey, req.ComputerName, assignedIP)
		if err != nil {
			logger.Error("Error adding computer to network", "error", err)
			s.sendErrorSignal(conn, "Error adding computer to network", originalID)
			return
		}
	} else {
		// If already in network, retrieve existing IP
		computer, err := s.supabaseManager.GetComputerInNetwork(req.NetworkID, req.PublicKey)
		if err != nil {
			logger.Error("Error getting computer from network", "error", err)
			s.sendErrorSignal(conn, "Error retrieving existing IP", originalID)
			return
		}
		assignedIP = computer.PeerIP

		// Update connection status in memory
		if _, ok := s.connectedPeers[req.NetworkID]; !ok {
			s.connectedPeers[req.NetworkID] = make(map[string]bool)
		}
		s.connectedPeers[req.NetworkID][req.PublicKey] = true
	}

	s.clientToPublicKey[conn] = req.PublicKey

	s.clients[conn] = req.NetworkID
	if _, exists := s.networks[req.NetworkID]; !exists {
		s.networks[req.NetworkID] = []*websocket.Conn{}
	}
	s.networks[req.NetworkID] = append(s.networks[req.NetworkID], conn)

	clientCount := len(s.networks[req.NetworkID])

	err = s.supabaseManager.UpdateNetworkActivity(req.NetworkID)
	if err != nil && (s.config.LogLevel == "debug") {
		logger.Debug("Error updating network activity", "error", err)
	}

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		logger.Info("Client joined network",
			"clientAddr", conn.RemoteAddr().String(),
			"networkID", req.NetworkID,
			"activeClients", clientCount,
			"assignedIP", assignedIP)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	responsePayload := map[string]interface{}{
		"network_id":   req.NetworkID,
		"network_name": network.Name,
		"peer_ip":      assignedIP,
	}
	s.sendSignal(conn, models.TypeNetworkJoined, responsePayload, originalID)

	// Notify other clients in the network about the new peer
	for _, computer := range s.networks[req.NetworkID] {
		if computer != conn {
			computerJoinedPayload := map[string]interface{}{
				"network_id":   req.NetworkID,
				"public_key":   req.PublicKey,
				"computername": req.ComputerName,
				"peer_ip":      assignedIP,
			}
			s.sendSignal(computer, models.TypeComputerJoined, computerJoinedPayload, "")
		}
	}

	// Send existing peers' info to the newly joined client
	for _, existingConn := range s.networks[req.NetworkID] {
		if existingConn != conn {
			existingPublicKey, hasExistingKey := s.clientToPublicKey[existingConn]
			if hasExistingKey {
				existingComputer, err := s.supabaseManager.GetComputerInNetwork(req.NetworkID, existingPublicKey)
				if err == nil {
					existingComputerPayload := map[string]interface{}{
						"network_id":   req.NetworkID,
						"public_key":   existingPublicKey,
						"computername": existingComputer.ComputerName,
						"peer_ip":      existingComputer.PeerIP,
					}
					s.sendSignal(conn, models.TypeComputerJoined, existingComputerPayload, "")
				}
			}
		}
	}
}

func (s *WebSocketServer) handleConnectNetwork(conn *websocket.Conn, req models.ConnectNetworkRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	network, err := s.supabaseManager.GetNetwork(req.NetworkID)
	if err != nil {
		s.sendErrorSignal(conn, "Network does not exist", originalID)
		return
	}

	if req.PublicKey == "" {
		s.sendErrorSignal(conn, "Public key is required", originalID)
		return
	}

	computer, err := s.supabaseManager.GetComputerInNetwork(req.NetworkID, req.PublicKey)
	if err != nil {
		logger.Error("Error getting computer from network", "error", err)
		s.sendErrorSignal(conn, "You must join this network first", originalID)
		return
	}

	connections := s.networks[req.NetworkID]
	if len(connections) >= s.config.MaxClientsPerNetwork {
		s.sendErrorSignal(conn, "Network is full", originalID)
		return
	}

	// Update connection status in memory
	if _, ok := s.connectedPeers[req.NetworkID]; !ok {
		s.connectedPeers[req.NetworkID] = make(map[string]bool)
	}
	s.connectedPeers[req.NetworkID][req.PublicKey] = true

	s.clientToPublicKey[conn] = req.PublicKey

	s.clients[conn] = req.NetworkID
	if _, exists := s.networks[req.NetworkID]; !exists {
		s.networks[req.NetworkID] = []*websocket.Conn{}
	}
	s.networks[req.NetworkID] = append(s.networks[req.NetworkID], conn)

	err = s.supabaseManager.UpdateNetworkActivity(req.NetworkID)
	if err != nil && (s.config.LogLevel == "debug") {
		logger.Debug("Error updating network activity", "error", err)
	}

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		logger.Info("Client connected to network (reconnect)",
			"clientAddr", conn.RemoteAddr().String(),
			"networkID", req.NetworkID,
			"activeClients", len(s.networks[req.NetworkID]),
			"assignedIP", computer.PeerIP)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	responsePayload := map[string]interface{}{
		"network_id":   req.NetworkID,
		"network_name": network.Name,
		"peer_ip":      computer.PeerIP,
	}
	s.sendSignal(conn, models.TypeNetworkConnected, responsePayload, originalID)

	// Notify other clients in the network about the new peer
	for _, computerConn := range s.networks[req.NetworkID] {
		if computerConn != conn {
			computerConnectedPayload := map[string]interface{}{
				"network_id":   req.NetworkID,
				"public_key":   req.PublicKey,
				"computername": req.ComputerName,
				"peer_ip":      computer.PeerIP,
			}
			s.sendSignal(computerConn, models.TypeComputerConnected, computerConnectedPayload, "")
		}
	}

	// Send existing peers' info to the newly connected client
	for _, existingConn := range s.networks[req.NetworkID] {
		if existingConn != conn {
			existingPublicKey, hasExistingKey := s.clientToPublicKey[existingConn]
			if hasExistingKey {
				existingComputer, err := s.supabaseManager.GetComputerInNetwork(req.NetworkID, existingPublicKey)
				if err == nil {
					existingComputerPayload := map[string]interface{}{
						"network_id":   req.NetworkID,
						"public_key":   existingPublicKey,
						"computername": existingComputer.ComputerName,
						"peer_ip":      existingComputer.PeerIP,
					}
					s.sendSignal(conn, models.TypeComputerConnected, existingComputerPayload, "")
				}
			}
		}
	}
}

func (s *WebSocketServer) handleDisconnectNetwork(conn *websocket.Conn, req models.DisconnectNetworkRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	networkID := req.NetworkID

	if networkID == "" {
		networkID = s.clients[conn]
		if networkID == "" {
			s.sendErrorSignal(conn, "Not connected to any network", originalID)
			return
		}
	}

	if s.clients[conn] != networkID {
		s.sendErrorSignal(conn, "Not connected to this network", originalID)
		return
	}

	publicKey, hasPublicKey := s.clientToPublicKey[conn]
	if !hasPublicKey {
		s.sendErrorSignal(conn, "Public key not found for this connection", originalID)
		return
	}

	network, err := s.supabaseManager.GetNetwork(networkID)
	if err != nil {
		s.sendErrorSignal(conn, "Network not found", originalID)
		return
	}

	isOwner := (publicKey == network.PublicKeyB64)

	if isOwner {
		err = s.supabaseManager.UpdateNetworkActivity(networkID)
		if err != nil && (s.config.LogLevel == "debug") {
			logger.Debug("Error updating network activity", "error", err)
		}
	}

	// Update connection status in memory
	if peers, ok := s.connectedPeers[networkID]; ok {
		delete(peers, publicKey)
		if len(peers) == 0 {
			delete(s.connectedPeers, networkID)
		}
	}

	if networks, exists := s.networks[networkID]; exists {
		for i, computer := range networks {
			if computer == conn {
				s.networks[networkID] = append(networks[:i], networks[i+1:]...)
				break
			}
		}

		if len(s.networks[networkID]) == 0 {
			delete(s.networks, networkID)
		} else {
			for _, computer := range s.networks[networkID] {
				computerDisconnectedPayload := map[string]interface{}{
					"network_id": networkID,
					"public_key": publicKey,
				}
				s.sendSignal(computer, models.TypeComputerDisconnected, computerDisconnectedPayload, "")
			}
		}
	}

	disconnectResponse := map[string]interface{}{
		"network_id": networkID,
	}
	s.sendSignal(conn, models.TypeNetworkDisconnected, disconnectResponse, originalID)

	if s.config.LogLevel == "info" || s.config.LogLevel == "debug" {
		logger.Info("Client disconnected from network (but still a member)",
			"clientAddr", conn.RemoteAddr().String(),
			"networkID", networkID,
			"isOwner", isOwner)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))
}

func (s *WebSocketServer) handleLeaveNetwork(conn *websocket.Conn, req models.LeaveNetworkRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	networkID := req.NetworkID

	if networkID == "" {
		networkID = s.clients[conn]
		if networkID == "" {
			s.sendErrorSignal(conn, "Not connected to any network", originalID)
			return
		}
	}

	publicKey := req.PublicKey
	if publicKey == "" {
		var ok bool
		publicKey, ok = s.clientToPublicKey[conn]
		if !ok || publicKey == "" {
			s.sendErrorSignal(conn, "Public key is required", originalID)
			return
		}
	}

	network, err := s.supabaseManager.GetNetwork(networkID)
	if err != nil {
		s.sendErrorSignal(conn, "Network not found", originalID)
		return
	}

	isCreator := (publicKey == network.PublicKeyB64)

	err = s.supabaseManager.RemoveComputerFromNetwork(networkID, publicKey)
	if err != nil {
		logger.Error("Error removing computer from computer_networks table", "error", err)
	}

	if isCreator {
		logger.Info("Network owner leaving", "networkID", networkID, "intentionalDelete", true)

		for _, computer := range s.networks[networkID] {
			if computer != conn {
				deletedNotification := models.NetworkDeletedNotification{
					NetworkID: networkID,
				}
				s.sendSignal(computer, models.TypeNetworkDeleted, deletedNotification, "")
			}
		}

		err := s.supabaseManager.DeleteNetwork(networkID)
		if err != nil {
			logger.Error("Error deleting network from Supabase", "error", err)
		}

		delete(s.networks, networkID)
		for c, cNetworkID := range s.clients {
			if cNetworkID == networkID {
				delete(s.clients, c)
			}
		}

		logger.Info("Network deleted because owner left", "networkID", networkID)
	} else {
		s.removeClient(conn, networkID)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	leaveSuccessPayload := map[string]interface{}{
		"network_id": networkID,
	}
	s.sendSignal(conn, models.TypeLeaveNetwork, leaveSuccessPayload, originalID)

	logger.Info("Client left network via explicit leave",
		"clientAddr", conn.RemoteAddr().String(),
		"networkID", networkID,
		"isCreator", isCreator)
}

// handleKick processes a request to kick a computer from the network
func (s *WebSocketServer) handleKick(conn *websocket.Conn, req models.KickRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	network, err := s.supabaseManager.GetNetwork(req.NetworkID)
	if err != nil {
		s.sendErrorSignal(conn, "Network does not exist", originalID)
		return
	}

	// Verifica se o cliente é o dono da sala
	publicKey, hasPublicKey := s.clientToPublicKey[conn]
	if !hasPublicKey || publicKey != network.PublicKeyB64 {
		s.sendErrorSignal(conn, "Only network owner can kick computers", originalID)
		return
	}

	for _, computer := range s.networks[req.NetworkID] {
		if computer.RemoteAddr().String() == req.TargetID {
			kickedPayload := map[string]interface{}{
				"network_id": req.NetworkID,
			}
			s.sendSignal(computer, models.TypeKicked, kickedPayload, "")

			computer.Close()
			s.removeClient(computer, req.NetworkID)
			logger.Info("Client kicked from network", "targetID", req.TargetID, "networkID", req.NetworkID)

			s.statsManager.UpdateStats(len(s.clients), len(s.networks))

			kickSuccessPayload := map[string]interface{}{
				"network_id": req.NetworkID,
				"target_id":  req.TargetID,
			}
			s.sendSignal(conn, models.TypeKickSuccess, kickSuccessPayload, originalID)
			return
		}
	}

	s.sendErrorSignal(conn, "Target client not found", originalID)
}

// handleRename processes a request to rename a network
func (s *WebSocketServer) handleRename(conn *websocket.Conn, req models.RenameRequest, originalID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	network, err := s.supabaseManager.GetNetwork(req.NetworkID)
	if err != nil {
		s.sendErrorSignal(conn, "Network does not exist", originalID)
		return
	}

	// Verifica se o cliente é o dono da sala
	publicKey, hasPublicKey := s.clientToPublicKey[conn]
	if !hasPublicKey || publicKey != network.PublicKeyB64 {
		s.sendErrorSignal(conn, "Only network owner can rename the network", originalID)
		return
	}

	err = s.supabaseManager.UpdateNetworkName(req.NetworkID, req.NetworkName)
	if err != nil {
		logger.Error("Error updating network name", "error", err)
		s.sendErrorSignal(conn, "Error updating network name in database", originalID)
		return
	}

	logger.Info("Network renamed", "networkID", req.NetworkID, "newName", req.NetworkName)

	// Notify all clients in the network about the rename
	renamePayload := map[string]interface{}{
		"network_id":   req.NetworkID,
		"network_name": req.NetworkName,
	}

	for _, computer := range s.networks[req.NetworkID] {
		s.sendSignal(computer, models.TypeNetworkRenamed, renamePayload, "")
	}

	// Additional successful rename notification to the requester
	s.sendSignal(conn, models.TypeRenameSuccess, renamePayload, originalID)
}

// handleDisconnect manages cleanup when a client disconnects
// Logic: Remove client from network tracking, update Supabase for network owners, clean up resources
func (s *WebSocketServer) handleDisconnect(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Quando ocorre uma desconexão (fechamento do app ou perda de conexão),
	networkID := s.clients[conn]
	if networkID != "" {
		// Recupera a chave pública do cliente antes de removê-lo
		publicKey, hasPublicKey := s.clientToPublicKey[conn]
		isOwner := false

		// Verifica se o cliente que está saindo é o dono da sala com base na chave pública
		if hasPublicKey {
			// Busca a sala no banco de dados
			network, err := s.supabaseManager.GetNetwork(networkID)
			if err == nil && publicKey == network.PublicKeyB64 {
				// A chave pública do cliente que está saindo coincide com a do criador da sala
				isOwner = true
				logger.Info("Network owner disconnected",
					"keyPrefix", publicKey[:10]+"...")

				// Update network activity to prevent it from being removed during cleanup
				err := s.supabaseManager.UpdateNetworkActivity(networkID)
				if err != nil {
					logger.Error("Error updating network activity", "error", err)
				}
			}
		}

		// Notifica outros membros da sala sobre a saída deste cliente
		for _, computer := range s.networks[networkID] {
			if computer != conn {
				computerLeftPayload := map[string]interface{}{
					"network_id": networkID,
					"public_key": publicKey,
				}
				s.sendSignal(computer, models.TypeComputerLeft, computerLeftPayload, "")
			}
		}

		// Limpa as referências do cliente
		delete(s.clients, conn)
		delete(s.clientToPublicKey, conn)

		// Update connection status in memory
		if peers, ok := s.connectedPeers[networkID]; ok {
			delete(peers, publicKey)
			if len(peers) == 0 {
				delete(s.connectedPeers, networkID)
			}
		}

		// Se a sala não existe no networks, não há nada mais a fazer
		network, exists := s.networks[networkID]
		if !exists {
			return
		}

		// Remove o cliente da lista de conexões da sala
		for i, computer := range network {
			if computer == conn {
				s.networks[networkID] = append(network[:i], network[i+1:]...)
				break
			}
		}

		// Se não houver mais clientes na sala, remove apenas da memória mas mantém no banco
		if len(s.networks[networkID]) == 0 {
			delete(s.networks, networkID)
		}

		if isOwner {
			logger.Info("Owner disconnected but network preserved",
				"clientAddr", conn.RemoteAddr().String(),
				"networkID", networkID)
		} else {
			logger.Info("Client disconnected from network",
				"clientAddr", conn.RemoteAddr().String(),
				"networkID", networkID)
		}
	} else {
		// Cliente não estava em nenhuma sala
		delete(s.clientToPublicKey, conn)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))
}

// DeleteStaleNetworks removes networks that have not been active for a specified period
// Logic: Query for networks that haven't been active past the expiry period and delete them
func (s *WebSocketServer) DeleteStaleNetworks() {
	staleNetworks, err := s.supabaseManager.GetStaleNetworks(s.config.NetworkExpiryDays)
	if err != nil {
		logger.Error("Error fetching stale networks", "error", err)
		return
	}

	numRemoved := 0
	for _, network := range staleNetworks {
		err := s.supabaseManager.DeleteNetwork(network.ID)
		if err != nil {
			logger.Error("Error deleting stale network", "networkID", network.ID, "error", err)
		} else {
			logger.Info("Deleted stale network", "networkID", network.ID)
			numRemoved++
		}
	}

	s.statsManager.UpdateCleanupStats(numRemoved)
}

// Start initializes and starts the WebSocket server
// Logic: Set up HTTP handlers, start network cleanup routine, and listen for incoming connections
func (s *WebSocketServer) Start(port string) error {
	mux := http.NewServeMux()

	// Add handlers to the mux
	mux.HandleFunc("/ws", s.HandleWebSocketEndpoint)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Add stats endpoint
	mux.HandleFunc("/stats", s.handleStatsEndpoint)

	// Create an HTTP server with the mux
	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Periodically delete stale networks
	go func() {
		ticker := time.NewTicker(s.config.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.DeleteStaleNetworks()
			case <-s.shutdownChan:
				return // Stop cleanup goroutine when server shuts down
			}
		}
	}()

	logger.Info("WebSocket server starting", "port", port)

	// Start HTTP server in a separate goroutine so we can return errors
	go func() {
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// handlePing processes ping messages from clients and responds with a pong
// This allows clients to verify their connection to the server
func (s *WebSocketServer) handlePing(conn *websocket.Conn, payload []byte, originalID string) {
	// Parse the ping message
	var pingData map[string]interface{}
	if err := json.Unmarshal(payload, &pingData); err != nil {
		logger.Error("Error parsing ping payload", "error", err)
		s.sendErrorSignal(conn, "Invalid ping format", originalID)
		return
	}

	// Create response payload with server timestamp
	pongPayload := map[string]interface{}{
		"client_timestamp": pingData["timestamp"],
		"server_timestamp": time.Now().UnixNano(),
		"status":           "ok",
	}

	// Log ping if in debug mode
	if s.config.LogLevel == "debug" {
		clientAddr := conn.RemoteAddr().String()
		logger.Debug("Received ping from client", "clientAddr", clientAddr)
	}

	// Send pong response with the same message ID
	s.sendSignal(conn, models.TypePing, pongPayload, originalID)
}

// removeClient remove um cliente da sala e, se necessário, a sala do Supabase
// Logic: Clean up client references and potentially delete network if owner leaves
func (s *WebSocketServer) removeClient(conn *websocket.Conn, networkID string) {
	// Se o networkID não foi fornecido, não há nada a fazer
	if networkID == "" {
		// Limpa a chave pública do cliente
		delete(s.clientToPublicKey, conn)
		return
	}

	// Recupera a chave pública do cliente antes de removê-lo
	publicKey, hasPublicKey := s.clientToPublicKey[conn]

	// Limpa as referências do cliente
	delete(s.clients, conn)
	delete(s.clientToPublicKey, conn)

	// Se a sala não existe no networks, não há nada mais a fazer
	network, exists := s.networks[networkID]
	if !exists {
		return
	}

	// Remove o cliente da lista de conexões da sala
	for i, computer := range network {
		if computer == conn {
			s.networks[networkID] = append(network[:i], network[i+1:]...)
			break
		}
	}

	// Verifica se o cliente que está saindo é o dono da sala com base na chave pública
	isCreator := false
	if hasPublicKey {
		// Busca a sala no banco de dados
		network, err := s.supabaseManager.GetNetwork(networkID)
		if err == nil && publicKey == network.PublicKeyB64 {
			// A chave pública do cliente que está saindo coincide com a do criador da sala
			isCreator = true
			logger.Info("Network creator disconnected", "publicKey", publicKey)
		}
	}

	// Se for o criador, deleta a sala do banco de dados
	if isCreator {
		err := s.supabaseManager.DeleteNetwork(networkID)
		if err != nil && (s.config.LogLevel == "debug") {
			logger.Debug("Error deleting network from Supabase on creator disconnect", "error", err)
		}

		// Notifica todos os outros participantes da sala que ela foi excluída
		for _, computer := range s.networks[networkID] {
			if computer != conn {
				// Usando o struct correto do models para TypeNetworkDeleted
				deletedNotification := models.NetworkDeletedNotification{
					NetworkID: networkID,
				}
				s.sendSignal(computer, models.TypeNetworkDeleted, deletedNotification, "")
			}
		}

		// Remove completamente a sala e seus clientes
		delete(s.networks, networkID)
		for c, cNetworkID := range s.clients {
			if cNetworkID == networkID {
				delete(s.clients, c)
			}
		}
		logger.Info("Network deleted because owner disconnected", "networkID", networkID)
	} else if len(s.networks[networkID]) == 0 {
		// Se não houver mais clientes na sala, remove a referência da sala
		delete(s.networks, networkID)
	}

	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	logger.Info("Client left network", "clientAddr", conn.RemoteAddr().String(), "networkID", networkID)
}

// handleStatsEndpoint is the HTTP handler for the /stats endpoint
func (s *WebSocketServer) handleStatsEndpoint(w http.ResponseWriter, r *http.Request) {
	s.statsManager.UpdateStats(len(s.clients), len(s.networks))

	w.Header().Set("Content-Type", "application/json")

	// Create response with current statistics
	statsResponse := map[string]interface{}{
		"server_stats": s.statsManager.GetStats(),
		"config": map[string]interface{}{
			"max_clients_per_network": s.config.MaxClientsPerNetwork,
			"network_expiry_days":     s.config.NetworkExpiryDays,
			"cleanup_interval":        s.config.CleanupInterval.String(),
			"allow_all_origins":       s.config.AllowAllOrigins,
		},
	}

	// Convert to JSON and send response
	jsonBytes, err := json.Marshal(statsResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to generate statistics"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

// handleGetComputerNetworks processes a request to get all networks a computer has joined
func (s *WebSocketServer) handleGetComputerNetworks(conn *websocket.Conn, req models.GetComputerNetworksRequest, originalID string) {
	s.handleGetComputerNetworksWithIP(conn, req, originalID, nil)
}

// handleGetComputerNetworksWithIP processes a request to get all networks a computer has joined and optionally sends IP info
func (s *WebSocketServer) handleGetComputerNetworksWithIP(conn *websocket.Conn, req models.GetComputerNetworksRequest, originalID string, httpReq *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if req.PublicKey == "" {
		s.sendErrorSignal(conn, "Public key is required", originalID)
		return
	}

	// Get all networks for this computer
	computerNetworks, err := s.supabaseManager.GetComputerNetworks(req.PublicKey)
	if err != nil {
		logger.Error("Error fetching computer networks", "error", err, "publicKey", req.PublicKey)
		s.sendErrorSignal(conn, "Error fetching computer networks", originalID)
		return
	}

	// Build response with network details
	response := models.ComputerNetworksResponse{
		Networks: make([]models.ComputerNetworkInfo, 0, len(computerNetworks)),
	}

	for _, computerNetwork := range computerNetworks {
		// Get network details
		network, err := s.supabaseManager.GetNetwork(computerNetwork.NetworkID)
		if err != nil {
			// Skip networks that no longer exist
			logger.Debug("Network no longer exists", "networkID", computerNetwork.NetworkID)
			continue
		}

		networkInfo := models.ComputerNetworkInfo{
			NetworkID:     computerNetwork.NetworkID,
			NetworkName:   network.Name,
			JoinedAt:      computerNetwork.JoinedAt,
			LastConnected: computerNetwork.LastConnected,
		}
		response.Networks = append(response.Networks, networkInfo)
	}

	if s.config.LogLevel == "debug" {
		logger.Debug("Sending computer networks",
			"publicKey", req.PublicKey,
			"networkCount", len(response.Networks))
	}

	// Send networks response
	s.sendSignal(conn, models.TypeComputerNetworks, response, originalID)

	// If HTTP request is provided, also send client IP information
	if httpReq != nil {
		ipInfo := s.getClientIPInfo(httpReq)
		// Generate a new message ID for the IP info
		ipMsgID, _ := models.GenerateMessageID()
		s.sendSignal(conn, models.TypeClientIPInfo, ipInfo, ipMsgID)
	}
}

// InitiateGracefulShutdown starts the graceful shutdown process
func (s *WebSocketServer) InitiateGracefulShutdown(timeout time.Duration, restartInfo string) {
	if s.isShutdown {
		logger.Warn("Shutdown already in progress")
		return
	}

	s.isShutdown = true
	logger.Info("Initiating graceful shutdown", "timeout", timeout)

	// Wait a short amount of time for in-flight requests to complete
	time.Sleep(500 * time.Millisecond)

	// First step: Notify all connected clients about impending shutdown
	s.notifyClientsAboutShutdown(int(timeout.Seconds()), restartInfo)

	// Allow some time for notification messages to be sent
	time.Sleep(1 * time.Second)

	// Set a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Start persisting server state
	s.persistStateForRestart()

	// Actually shutdown the HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			logger.Error("HTTP server shutdown error", "error", err)
		}
	}

	// Signal successful shutdown
	close(s.shutdownChan)
	logger.Info("Graceful shutdown completed")
}

// notifyClientsAboutShutdown sends a shutdown notification to all connected clients
func (s *WebSocketServer) notifyClientsAboutShutdown(shutdownSeconds int, restartInfo string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logger.Info("Notifying clients about server shutdown", "clientCount", len(s.clients))

	notification := models.ServerShutdownNotification{
		Message:     "Server is shutting down for maintenance",
		ShutdownIn:  shutdownSeconds,
		RestartInfo: restartInfo,
	}

	// Send notification to all clients
	for conn := range s.clients {
		// Generate a random message ID
		msgID, err := models.GenerateMessageID()
		if err != nil {
			msgID = ""
		}

		err = s.sendSignal(conn, models.TypeServerShutdown, notification, msgID)
		if err != nil {
			logger.Error("Error notifying client", "clientAddr", conn.RemoteAddr().String(), "error", err)
		}
	}
}

// persistStateForRestart saves the current server state to enable a clean restart
func (s *WebSocketServer) persistStateForRestart() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logger.Info("Persisting state for potential server restart")

	// Update all active network timestamps in the database
	for networkID := range s.networks {
		// Ensure this network's activity is updated to prevent cleanup
		err := s.supabaseManager.UpdateNetworkActivity(networkID)
		if err != nil {
			logger.Error("Error updating network activity", "networkID", networkID, "error", err)
		} else {
			logger.Info("Network state persisted", "networkID", networkID)
		}
	}

	// Could add more state persistence here if needed
	logger.Info("Server state persistence completed")
}

// WaitForShutdown blocks until the server has shut down
func (s *WebSocketServer) WaitForShutdown() {
	<-s.shutdownChan
}

// extractIP extracts the client's IP address from the request
// getClientIPInfo extracts IPv4 and IPv6 addresses from the client request
func (s *WebSocketServer) getClientIPInfo(r *http.Request) models.ClientIPInfoResponse {
	ipInfo := models.ClientIPInfoResponse{}

	// Get the client's remote address
	remoteAddr := r.RemoteAddr

	// Check for X-Forwarded-For header (common in proxied environments)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			remoteAddr = strings.TrimSpace(ips[0])
		}
	}

	// Check for X-Real-IP header (another common proxy header)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		remoteAddr = realIP
	}

	// Parse the IP address
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If SplitHostPort fails, assume the remoteAddr is just an IP
		host = remoteAddr
	}

	// Parse the IP to determine if it's IPv4 or IPv6
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.To4() != nil {
			// It's an IPv4 address
			ipInfo.IPv4 = ip.String()
		} else {
			// It's an IPv6 address
			ipInfo.IPv6 = ip.String()
		}
	}

	return ipInfo
}
