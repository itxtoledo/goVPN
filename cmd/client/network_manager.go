package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"github.com/itxtoledo/govpn/cmd/client/data"
	
	sclient "github.com/itxtoledo/govpn/libs/signaling/client"
	smodels "github.com/itxtoledo/govpn/libs/signaling/models"
	"github.com/pion/webrtc/v4"
	clientwebrtc_impl "github.com/itxtoledo/govpn/cmd/client/webrtc"
)

// NetworkInterface define a interface mínima para a rede virtual
type NetworkInterface interface {
	GetLocalIP() string
	GetComputerCount() int
}

// ConnectionState represents the state of the connection
type ConnectionState int

const (
	ConnectionStateDisconnected ConnectionState = iota
	ConnectionStateConnecting
	ConnectionStateConnected
)

// NetworkManager handles the VPN network
// OnWebRTCMessageReceived is a callback function for incoming WebRTC messages
type OnWebRTCMessageReceived func(peerPublicKey string, message string)

// NetworkManager handles the VPN network
type NetworkManager struct {
	peerConnections map[string]*clientwebrtc_impl.WebRTCManager // Map of peer public key to their WebRTC manager

	VirtualNetwork    NetworkInterface
	SignalingServer   *sclient.SignalingClient
	NetworkID         string
	connectionState   ConnectionState
	ReconnectAttempts int
	MaxReconnects     int

	// Dependencies
	RealtimeData            *data.RealtimeDataLayer
	ConfigManager           *ConfigManager
	refreshNetworkList      func()
	refreshUI               func()
	onWebRTCMessageReceived OnWebRTCMessageReceived
}

// NewNetworkManager creates a new instance of NetworkManager
func NewNetworkManager(realtimeData *data.RealtimeDataLayer, configManager *ConfigManager, refreshNetworkList func(), refreshUI func(), onWebRTCMessageReceived OnWebRTCMessageReceived) *NetworkManager {
	nm := &NetworkManager{
		peerConnections:    make(map[string]*clientwebrtc_impl.WebRTCManager),
		connectionState:    ConnectionStateDisconnected,
		ReconnectAttempts:  0,
		MaxReconnects:      5,
		RealtimeData:       realtimeData,
		ConfigManager:      configManager,
		refreshNetworkList: refreshNetworkList,
		refreshUI:          refreshUI,
		onWebRTCMessageReceived: onWebRTCMessageReceived,
	}

	return nm
}

// Connect connects to the VPN network
func (nm *NetworkManager) Connect(serverAddress string) error {
	// Set state to connecting
	nm.connectionState = ConnectionStateConnecting
	// Update data layer
	nm.RealtimeData.SetConnectionState(data.StateConnecting)
	nm.RealtimeData.SetStatusMessage("Connecting...")

	// Update UI
	nm.refreshUI()

	// Initialize signaling server
	// Get public key from ConfigManager
	publicKey, _ := nm.ConfigManager.GetKeyPair()

	// Create a handler function for signaling client messages
	signalingHandler := func(messageType smodels.MessageType, payload []byte) {
		switch messageType {
		case smodels.TypeError:
			var errorPayload map[string]string
			if err := json.Unmarshal(payload, &errorPayload); err == nil {
				if errorMsg, ok := errorPayload["error"]; ok {
					log.Printf("Server error: %s", errorMsg)
					nm.RealtimeData.EmitEvent(data.EventError, errorMsg, nil)
				}
			}
		case smodels.TypeNetworkDisconnected:
			var networkDisconnectedResponse smodels.DisconnectNetworkResponse
			if err := json.Unmarshal(payload, &networkDisconnectedResponse); err != nil {
				log.Printf("Failed to unmarshal network disconnected response: %v", err)
				return
			}

			// Find the network in RealtimeData and update its LastConnected time
			networks := nm.RealtimeData.GetNetworks()
			for i, network := range networks {
				if network.NetworkID == networkDisconnectedResponse.NetworkID {
					network.LastConnected = time.Now() // Update last connected time
					nm.RealtimeData.UpdateNetwork(i, network)
					break
				}
			}
			nm.refreshNetworkList()
		case smodels.TypeNetworkJoined:
			nm.refreshNetworkList()
		case smodels.TypeNetworkCreated:
			var createNetworkResponse smodels.CreateNetworkResponse
			if err := json.Unmarshal(payload, &createNetworkResponse); err != nil {
				log.Printf("Failed to unmarshal create network response: %v", err)
				return
			}
			fyne.Do(func() {
				network := data.Network{
					NetworkID:   createNetworkResponse.NetworkID,
					NetworkName: createNetworkResponse.NetworkName,
					Computers:   createNetworkResponse.Computers,
					LastConnected: time.Now(),
				}
				nm.RealtimeData.AddNetwork(network)
			})
		case smodels.TypeLeaveNetwork:
			nm.refreshNetworkList()
		case smodels.TypeKicked:
			nm.refreshNetworkList()
		case smodels.TypeComputerJoined:
			var computerJoinedNotification smodels.ComputerJoinedNotification
			if err := json.Unmarshal(payload, &computerJoinedNotification); err != nil {
				log.Printf("Failed to unmarshal computer joined notification: %v", err)
				return
			}

			log.Printf("Computer %s (IP: %s) joined network %s", computerJoinedNotification.ComputerName, computerJoinedNotification.ComputerIP, computerJoinedNotification.NetworkID)

			// Find the network and add the new computer
			networks := nm.RealtimeData.GetNetworks()
			for i, network := range networks {
				if network.NetworkID == computerJoinedNotification.NetworkID {
					// Check if computer already exists to avoid duplicates
					computerExists := false
					for _, computer := range network.Computers {
						if computer.PublicKey == computerJoinedNotification.PublicKey {
							computerExists = true
							break
						}
					}

					if !computerExists {
						network.Computers = append(network.Computers, smodels.ComputerInfo{
							Name:       computerJoinedNotification.ComputerName,
							ComputerIP: computerJoinedNotification.ComputerIP,
							PublicKey:  computerJoinedNotification.PublicKey,
						})
						nm.RealtimeData.UpdateNetwork(i, network)
						log.Printf("Added computer %s to network %s", computerJoinedNotification.ComputerName, network.NetworkName)
					}
					break
				}
			}
			nm.refreshNetworkList()
		case smodels.TypeComputerLeft:
			var computerLeftNotification smodels.ComputerLeftNotification
			if err := json.Unmarshal(payload, &computerLeftNotification); err != nil {
				log.Printf("Failed to unmarshal computer left notification: %v", err)
				return
			}

			log.Printf("Computer with public key %s left network %s", computerLeftNotification.PublicKey, computerLeftNotification.NetworkID)

			// Find the network and remove the computer
			networks := nm.RealtimeData.GetNetworks()
			for i, network := range networks {
				if network.NetworkID == computerLeftNotification.NetworkID {
					updatedComputers := []smodels.ComputerInfo{}
					for _, computer := range network.Computers {
						if computer.PublicKey != computerLeftNotification.PublicKey {
							updatedComputers = append(updatedComputers, computer)
						}
					}
					network.Computers = updatedComputers
					nm.RealtimeData.UpdateNetwork(i, network)
					log.Printf("Removed computer with public key %s from network %s", computerLeftNotification.PublicKey, network.NetworkName)
					break
				}
			}
			nm.refreshNetworkList()
		case smodels.TypeComputerNetworks:
			// For TypeComputerNetworks, we need to unmarshal the payload to update the networks list
			var computerNetworksResponse smodels.ComputerNetworksResponse
			if err := json.Unmarshal(payload, &computerNetworksResponse); err != nil {
				log.Printf("Failed to unmarshal computer networks response in handler: %v", err)
				return
			}

			log.Println("Received ComputerNetworks update:")
			for _, network := range computerNetworksResponse.Networks {
				log.Printf("  Network: %s (ID: %s)", network.NetworkName, network.NetworkID)
				for _, computer := range network.Computers {
					log.Printf("    Computer: %s (IP: %s)", computer.Name, computer.ComputerIP)
				}
			}

			// Update the RealtimeDataLayer with the new networks list
			nm.RealtimeData.SetNetworks(computerNetworksResponse.Networks)
			nm.refreshNetworkList()
		case smodels.TypeComputerConnected:
			var notification smodels.ComputerConnectedNotification
			if err := json.Unmarshal(payload, &notification); err != nil {
				log.Printf("Failed to unmarshal computer connected notification: %v", err)
				return
			}

			log.Printf("Computer %s (IP: %s) connected to network %s", notification.ComputerName, notification.ComputerIP, notification.NetworkID)

			// Find the network and update the computer's online status
			networks := nm.RealtimeData.GetNetworks()
			for i, network := range networks {
				if network.NetworkID == notification.NetworkID {
					for j, computer := range network.Computers {
						if computer.PublicKey == notification.PublicKey {
							network.Computers[j].IsOnline = true
							nm.RealtimeData.UpdateNetwork(i, network)
							log.Printf("Updated computer online status in UI for network %s", network.NetworkName)
							break
						}
					}
					break
				}
			}
			nm.refreshNetworkList()
		case smodels.TypeComputerDisconnected:
			var notification smodels.ComputerDisconnectedNotification
			if err := json.Unmarshal(payload, &notification); err != nil {
				log.Printf("Failed to unmarshal computer disconnected notification: %v", err)
				return
			}

			log.Printf("Computer with public key %s disconnected from network %s", notification.PublicKey, notification.NetworkID)

			// Find the network and update the computer's online status
			networks := nm.RealtimeData.GetNetworks()
			for i, network := range networks {
				if network.NetworkID == notification.NetworkID {
					for j, computer := range network.Computers {
						if computer.PublicKey == notification.PublicKey {
							network.Computers[j].IsOnline = false
							nm.RealtimeData.UpdateNetwork(i, network)
							log.Printf("Updated computer online status in UI for network %s", network.NetworkName)
							break
						}
					}
					break
				}
			}
			nm.refreshNetworkList()
		case smodels.TypeComputerRenamed:
			var notification smodels.ComputerRenamedNotification
			if err := json.Unmarshal(payload, &notification); err != nil {
				log.Printf("Failed to unmarshal computer renamed notification: %v", err)
				return
			}

			log.Printf("Computer %s in network %s renamed to %s", notification.PublicKey, notification.NetworkID, notification.NewComputerName)

			// Find the network and update the computer's name
			networks := nm.RealtimeData.GetNetworks()
			for i, network := range networks {
				if network.NetworkID == notification.NetworkID {
					for j, computer := range network.Computers {
						if computer.PublicKey == notification.PublicKey {
							network.Computers[j].Name = notification.NewComputerName
							nm.RealtimeData.UpdateNetwork(i, network)
							log.Printf("Updated computer name in UI for network %s", network.NetworkName)
							break
						}
					}
					break
				}
			}
			nm.refreshNetworkList()
		case smodels.TypeSdpOffer:
			var offer smodels.SdpOffer
			if err := json.Unmarshal(payload, &offer); err != nil {
				log.Printf("failed to unmarshal sdp offer: %v", err)
				return
			}

			// Get or create WebRTCManager for this peer
			peerWebRTCManager, ok := nm.peerConnections[offer.SenderPublicKey]
			if !ok {
				log.Printf("Creating new WebRTCManager for peer %s on receiving offer.", offer.SenderPublicKey)
				var err error // Declare err here
				peerWebRTCManager, err = clientwebrtc_impl.NewWebRTCManager()
				if err != nil {
					log.Printf("failed to create WebRTC manager for peer %s: %v", offer.SenderPublicKey, err)
					return
				}
				nm.peerConnections[offer.SenderPublicKey] = peerWebRTCManager

				// Set up callbacks for this specific peer connection
				peerWebRTCManager.SetOnICECandidate(func(c *webrtc.ICECandidate) {
					nm.handleICECandidate(c, offer.SenderPublicKey)
				})
				peerWebRTCManager.SetOnConnectionStateChange(func(s webrtc.PeerConnectionState) {
					nm.handlePeerConnectionStateChange(offer.SenderPublicKey, s)
				})
				peerWebRTCManager.SetOnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
					nm.handlePeerICEConnectionStateChange(offer.SenderPublicKey, s)
				})
				peerWebRTCManager.SetOnDataChannelOpen(func() {
					nm.handlePeerDataChannelOpen(offer.SenderPublicKey)
				})
				peerWebRTCManager.SetOnDataChannelMessage(func(msg []byte) {
					nm.handlePeerDataChannelMessage(offer.SenderPublicKey, msg)
				})

				// Create Data Channel for this peer if it's the answerer
				if err := peerWebRTCManager.CreateDataChannel(); err != nil {
					log.Printf("failed to create data channel for peer %s: %v", offer.SenderPublicKey, err)
					return
				}
			}

			answer, err := peerWebRTCManager.HandleOfferAndCreateAnswer(webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  offer.SDP,
			})
			if err != nil {
				log.Printf("failed to handle offer and create answer for peer %s: %v", offer.SenderPublicKey, err)
				return
			}

			_, err = nm.SignalingServer.SendMessage(smodels.TypeSdpAnswer, smodels.SdpAnswer{
				TargetPublicKey: offer.SenderPublicKey,
				SDP:             answer.SDP,
			})
			if err != nil {
				log.Printf("failed to send sdp answer for peer %s: %v", offer.SenderPublicKey, err)
				return
			}
		case smodels.TypeSdpAnswer:
			var answer smodels.SdpAnswer
			if err := json.Unmarshal(payload, &answer); err != nil {
				log.Printf("failed to unmarshal sdp answer: %v", err)
				return
			}

			peerWebRTCManager, ok := nm.peerConnections[answer.SenderPublicKey]
			if !ok {
				log.Printf("No WebRTCManager found for peer %s on receiving answer.", answer.SenderPublicKey)
				return
			}

			if err := peerWebRTCManager.HandleAnswer(webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  answer.SDP,
			}); err != nil {
				log.Printf("failed to set remote description for peer %s: %v", answer.SenderPublicKey, err)
				return
			}
		case smodels.TypeIceCandidate:
			var candidate smodels.IceCandidate
			if err := json.Unmarshal(payload, &candidate); err != nil {
				log.Printf("failed to unmarshal ice candidate: %v", err)
				return
			}

			peerWebRTCManager, ok := nm.peerConnections[candidate.SenderPublicKey]
			if !ok {
				log.Printf("No WebRTCManager found for peer %s on receiving ICE candidate.", candidate.SenderPublicKey)
				return
			}

			if err := peerWebRTCManager.AddICECandidate(webrtc.ICECandidateInit{
					Candidate:     candidate.Candidate,
					SDPMid:        &candidate.SDPMid,
					SDPMLineIndex: &candidate.SDPMLineIndex,
				}); err != nil {
				log.Printf("failed to add ICE candidate for peer %s: %v", candidate.SenderPublicKey, err)
				return
			}
		}
	}
	nm.SignalingServer = sclient.NewSignalingClient(publicKey, signalingHandler)

	// Connect to signaling server
	err := nm.SignalingServer.Connect(serverAddress)
	if err != nil {
		nm.connectionState = ConnectionStateDisconnected
		nm.RealtimeData.SetConnectionState(data.StateDisconnected)
		nm.RealtimeData.SetStatusMessage("Connection failed")
		return fmt.Errorf("failed to connect to signaling server: %v", err)
	}

	// Set state to connected
	nm.connectionState = ConnectionStateConnected
	nm.RealtimeData.SetConnectionState(data.StateConnected)
	nm.RealtimeData.SetStatusMessage("Connected")

	// O servidor já envia automaticamente a lista de salas do usuário ao conectar,
	// então não precisamos chamar GetComputerNetworks explicitamente
	log.Println("Aguardando lista de salas do servidor...")

	return nil
}

// handleDisconnection handles disconnection from the server
func (nm *NetworkManager) handleDisconnection() {
	if nm.connectionState != ConnectionStateDisconnected {
		if nm.ReconnectAttempts < nm.MaxReconnects {
			nm.ReconnectAttempts++
			log.Printf("Disconnected from server, attempting to reconnect (%d/%d)", nm.ReconnectAttempts, nm.MaxReconnects)

			// Set state to connecting
			nm.connectionState = ConnectionStateConnecting
			nm.RealtimeData.SetConnectionState(data.StateConnecting)
			nm.RealtimeData.SetStatusMessage(fmt.Sprintf("Reconnecting (%d/%d)...", nm.ReconnectAttempts, nm.MaxReconnects))
			nm.refreshUI()

			// Try to reconnect
			err := nm.Connect(nm.SignalingServer.ServerAddress)
			if err != nil {
				log.Printf("Failed to reconnect: %v", err)
				// Set state to disconnected
				nm.connectionState = ConnectionStateDisconnected
				nm.RealtimeData.SetConnectionState(data.StateDisconnected)
				nm.RealtimeData.SetStatusMessage("Disconnected")
				nm.refreshUI()
			} else {
				// Successfully reconnected
				nm.ReconnectAttempts = 0
			}
		} else {
			log.Printf("Max reconnect attempts reached, giving up")
			// Set state to disconnected
			nm.connectionState = ConnectionStateDisconnected
			nm.RealtimeData.SetConnectionState(data.StateDisconnected)
			nm.RealtimeData.SetStatusMessage("Connection lost")
			nm.RealtimeData.SetComputerIP("0.0.0.0") // Clear the IP when connection is lost
			nm.refreshUI()
		}
	}
}

// GetConnectionState returns the connection state
func (nm *NetworkManager) GetConnectionState() ConnectionState {
	return nm.connectionState
}

// UpdateClientInfo envia as informações do cliente para o servidor
func (nm *NetworkManager) UpdateClientInfo() {
	if nm.connectionState != ConnectionStateConnected {
		log.Println("Cannot update client info: not connected to server")
		return
	}

	config := nm.ConfigManager.GetConfig()
	clientName := config.ComputerName

	log.Printf("Sending client info to server: %s", clientName)

	// Criar a mensagem
	msg := smodels.UpdateClientInfoRequest{
		ClientName: clientName,
		BaseRequest: smodels.BaseRequest{
			PublicKey: config.PublicKey,
		},
	}

	// Enviar a mensagem
	_, err := nm.SignalingServer.SendMessage(smodels.TypeUpdateClientInfo, msg)
	if err != nil {
		log.Printf("Failed to send client info: %v", err)
	}
}

// CreateNetwork creates a new network
func (nm *NetworkManager) CreateNetwork(name string, pin string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Create network
	res, err := nm.SignalingServer.CreateNetwork(name, pin, nm.ConfigManager.GetConfig().ComputerName)
	if err != nil {
		return fmt.Errorf("failed to create network: %v", err)
	}

	// Check if the network was successfully created and has a valid ID
	if res == nil || res.NetworkID == "" {
		return fmt.Errorf("failed to create network: invalid server response")
	}

	log.Printf("Network created: ID=%s, Name=%s", res.NetworkID, name)

	// Store network information for current connection
	nm.NetworkID = res.NetworkID

	// Update data layer
	nm.RealtimeData.SetNetworkInfo(res.NetworkID)
	nm.RealtimeData.EmitEvent(data.EventNetworkJoined, res.NetworkID, nil)

	// Update UI
	nm.refreshUI()

	return nil
}

// JoinNetwork joins a network
func (nm *NetworkManager) JoinNetwork(networkID string, pin string, computername string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Join network
	res, err := nm.SignalingServer.JoinNetwork(networkID, pin, computername)
	if err != nil {
		return fmt.Errorf("failed to join network: %v", err)
	}

	// Use networkName from the response
	networkName := res.NetworkName

	log.Printf("Network joined: ID=%s, Name=%s", networkID, networkName)

	// Store network information for current connection
	nm.NetworkID = networkID

	// Update data layer
	nm.RealtimeData.SetNetworkInfo(networkID)
	nm.RealtimeData.SetComputerIP(res.ComputerIP)
	nm.RealtimeData.EmitEvent(data.EventNetworkJoined, networkID, nil)

	// Update UI
	nm.refreshUI()

	return nil
}

// ConnectNetwork connects to a previously joined network
func (nm *NetworkManager) ConnectNetwork(networkID string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	// Connect to network
	res, err := nm.SignalingServer.ConnectNetwork(networkID, "Computer")
	if err != nil {
		return fmt.Errorf("failed to connect to network: %v", err)
	}

	// Use networkName from the response
	networkName := res.NetworkName

	// Find the network in memory to update lastConnected time
	networkExists := false
	for i, existingNetwork := range nm.RealtimeData.GetNetworks() {
		if existingNetwork.NetworkID == networkID {
			networkExists = true

			// Update lastConnected time
			existingNetwork.LastConnected = time.Now()
			nm.RealtimeData.UpdateNetwork(i, existingNetwork)
			break
		}
	}

	if !networkExists {
		return fmt.Errorf("network not found in local storage")
	}

	log.Printf("Network connected: ID=%s, Name=%s", networkID, networkName)

	// Store network information for current connection
	nm.NetworkID = networkID

	// Update data layer (without password since we don't store it)
	nm.RealtimeData.SetNetworkInfo(networkID)
	nm.RealtimeData.SetComputerIP(res.ComputerIP)
	nm.RealtimeData.EmitEvent(data.EventNetworkJoined, networkID, nil)

	// Refresh network list now that we have re-connected to the network
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

	return nil
}

// DisconnectNetwork disconnects from a network without leaving it
func (nm *NetworkManager) DisconnectNetwork(networkID string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	log.Printf("Disconnecting from network with ID: %s", networkID)

	// Disconnect from network
	_, err := nm.SignalingServer.DisconnectNetwork(networkID)
	if err != nil {
		log.Printf("Error from SignalingClient when disconnecting from network: %v", err)
		return fmt.Errorf("failed to disconnect from network: %v", err)
	}

	// If we're disconnecting from the current network, clear our network information
	if nm.NetworkID == networkID {
		nm.NetworkID = ""

		// Update data layer
		nm.RealtimeData.SetNetworkInfo("Not connected")
		nm.RealtimeData.SetComputerIP("0.0.0.0")
		nm.RealtimeData.EmitEvent(data.EventNetworkDisconnected, networkID, nil)
	}

	// Update the IsOnline status of the current computer in the RealtimeData.Networks list
	publicKey, _ := nm.ConfigManager.GetKeyPair()
	networks := nm.RealtimeData.GetNetworks()
	for i, network := range networks {
		if network.NetworkID == networkID {
			for j, computer := range network.Computers {
				if computer.PublicKey == publicKey {
					network.Computers[j].IsOnline = false
					nm.RealtimeData.UpdateNetwork(i, network)
					log.Printf("Updated current computer's online status to false for network %s", network.NetworkName)
					break
				}
			}
			break
		}
	}

	// Refresh the network list UI
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

	return nil
}

// LeaveNetwork leaves the current network
func (nm *NetworkManager) LeaveNetwork() error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	log.Printf("Leaving network with ID: %s", nm.NetworkID)

	// Store network ID before clearing
	networkID := nm.NetworkID

	// Leave network
	_, err := nm.SignalingServer.LeaveNetwork(networkID)
	if err != nil {
		log.Printf("Error from SignalingClient when leaving network: %v", err)
		return fmt.Errorf("failed to leave network: %v", err)
	}

	// Remove the network from memory
	nm.RealtimeData.RemoveNetwork(networkID)

	// Clear network information
	nm.NetworkID = ""

	// Update data layer
	nm.RealtimeData.SetNetworkInfo("Not connected")
	nm.RealtimeData.SetComputerIP("0.0.0.0")
	nm.RealtimeData.EmitEvent(data.EventNetworkLeft, networkID, nil)

	// Refresh the network list UI
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

	return nil
}

// LeaveNetworkById leaves a specific network by ID
func (nm *NetworkManager) LeaveNetworkById(networkID string) error {
	if nm.connectionState != ConnectionStateConnected {
		return fmt.Errorf("not connected to server")
	}

	log.Printf("Leaving network with ID: %s", networkID)

	// Leave network
	_, err := nm.SignalingServer.LeaveNetwork(networkID)
	if err != nil {
		log.Printf("Error from SignalingClient when leaving network: %v", err)
		return fmt.Errorf("failed to leave network: %v", err)
	}

	// Remove network from memory
	nm.RealtimeData.RemoveNetwork(networkID)

	// If we're leaving the current network, clear our network information
	if nm.NetworkID == networkID {
		nm.NetworkID = ""

		// Update data layer
		nm.RealtimeData.SetNetworkInfo("Not connected")
	}

	// Emit the network left event regardless
	nm.RealtimeData.EmitEvent(data.EventNetworkLeft, networkID, nil)

	// Refresh the network list UI
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

	return nil
}

// HandleNetworkDeleted handles when a network has been deleted
func (nm *NetworkManager) HandleNetworkDeleted(networkID string) error {
	log.Printf("Handling network deletion for ID: %s", networkID)

	// If we're in this network, clear our network data
	if nm.NetworkID == networkID {
		nm.NetworkID = ""

		// Update data layer
		nm.RealtimeData.SetNetworkInfo("Not connected")
	}

	// Remove network from memory
	nm.RealtimeData.RemoveNetwork(networkID)

	// Emit the event
	nm.RealtimeData.EmitEvent(data.EventNetworkDeleted, networkID, nil)

	// Refresh the network list UI
	nm.refreshNetworkList()

	// Update UI
	nm.refreshUI()

	return nil
}

// handleICECandidate handles a new ICE candidate
func (nm *NetworkManager) handleICECandidate(c *webrtc.ICECandidate, targetPublicKey string) {
	if c == nil {
		return
	}

	_, err := nm.SignalingServer.SendMessage(smodels.TypeIceCandidate, smodels.IceCandidate{
		TargetPublicKey: targetPublicKey,
		Candidate:       c.ToJSON().Candidate,
		SDPMid:          *c.ToJSON().SDPMid,
		SDPMLineIndex:   *c.ToJSON().SDPMLineIndex,
	})
	if err != nil {
		log.Printf("failed to send ice candidate: %v", err)
	}
}

// handlePeerConnectionStateChange handles changes in a peer's WebRTC connection state
func (nm *NetworkManager) handlePeerConnectionStateChange(peerPublicKey string, s webrtc.PeerConnectionState) {
	log.Printf("Peer %s Connection State has changed: %s", peerPublicKey, s.String())
	// TODO: Update UI or take action based on connection state
}

// handlePeerICEConnectionStateChange handles changes in a peer's ICE connection state
func (nm *NetworkManager) handlePeerICEConnectionStateChange(peerPublicKey string, s webrtc.ICEConnectionState) {
	log.Printf("Peer %s ICE Connection State has changed: %s", peerPublicKey, s.String())
	// TODO: Update UI or take action based on ICE connection state
}

// handlePeerDataChannelOpen handles the event when a data channel opens for a peer
func (nm *NetworkManager) handlePeerDataChannelOpen(peerPublicKey string) {
	log.Printf("Data channel opened for peer: %s", peerPublicKey)
	// TODO: Send initial messages or update UI
}

// handlePeerDataChannelMessage handles incoming data channel messages from a peer
func (nm *NetworkManager) handlePeerDataChannelMessage(peerPublicKey string, msg []byte) {
	log.Printf("Message from peer %s: %s", peerPublicKey, string(msg))
	if nm.onWebRTCMessageReceived != nil {
		nm.onWebRTCMessageReceived(peerPublicKey, string(msg))
	}
}

// ConnectToPeer initiates a WebRTC connection with a peer
func (nm *NetworkManager) ConnectToPeer(peerPublicKey string) error {
	// Check if a connection already exists for this peer
	if _, ok := nm.peerConnections[peerPublicKey]; ok {
		log.Printf("Connection to peer %s already exists.", peerPublicKey)
		return nil
	}

	// Create a new WebRTCManager for this peer
	peerWebRTCManager, err := clientwebrtc_impl.NewWebRTCManager()
	if err != nil {
		return fmt.Errorf("failed to create WebRTC manager for peer %s: %w", peerPublicKey, err)
	}

	nm.peerConnections[peerPublicKey] = peerWebRTCManager

	// Set up callbacks for this specific peer connection
	peerWebRTCManager.SetOnICECandidate(func(c *webrtc.ICECandidate) {
		nm.handleICECandidate(c, peerPublicKey)
	})
	peerWebRTCManager.SetOnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		nm.handlePeerConnectionStateChange(peerPublicKey, s)
	})
	peerWebRTCManager.SetOnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		nm.handlePeerICEConnectionStateChange(peerPublicKey, s)
	})
	peerWebRTCManager.SetOnDataChannelOpen(func() {
		nm.handlePeerDataChannelOpen(peerPublicKey)
	})
	peerWebRTCManager.SetOnDataChannelMessage(func(msg []byte) {
		nm.handlePeerDataChannelMessage(peerPublicKey, msg)
	})

	// Create Data Channel for this peer
	if err := peerWebRTCManager.CreateDataChannel(); err != nil {
		return fmt.Errorf("failed to create data channel for peer %s: %w", peerPublicKey, err)
	}

	// Create offer for this peer
	offer, err := peerWebRTCManager.CreateOffer(false)
	if err != nil {
		return fmt.Errorf("failed to create offer for peer %s: %w", peerPublicKey, err)
	}

	// Send offer via signaling server
	_, err = nm.SignalingServer.SendMessage(smodels.TypeSdpOffer, smodels.SdpOffer{
		TargetPublicKey: peerPublicKey,
		SDP:             offer.SDP,
	})
	if err != nil {
		return fmt.Errorf("failed to send sdp offer for peer %s: %w", peerPublicKey, err)
	}

	log.Printf("Initiated WebRTC connection with peer: %s", peerPublicKey)
	return nil
}

// Disconnect disconnects from the VPN network
func (nm *NetworkManager) Disconnect() error {
	if nm.connectionState == ConnectionStateDisconnected {
		return nil
	}

	// Explicitly disconnect from all joined networks before disconnecting from the signaling server
	// This ensures the server is notified of our disconnection from each network.
	networksToDisconnect := nm.RealtimeData.GetNetworks()
	for _, network := range networksToDisconnect {
		log.Printf("Explicitly disconnecting from network %s before full client disconnect.", network.NetworkID)
		err := nm.DisconnectNetwork(network.NetworkID)
		if err != nil {
			log.Printf("Error explicitly disconnecting from network %s: %v", network.NetworkID, err)
		}
	}

	// Disconnect from signaling server
	if nm.SignalingServer != nil {
		err := nm.SignalingServer.Disconnect()
		if err != nil {
			log.Printf("Error disconnecting from signaling server: %v", err)
		}
	}

	// Stop virtual network
	if nm.VirtualNetwork != nil {
		nm.VirtualNetwork = nil
	}

	// Close all WebRTC connections
	for peerPublicKey, peerWebRTCManager := range nm.peerConnections {
		if err := peerWebRTCManager.Close(); err != nil {
			log.Printf("Error closing WebRTC manager for peer %s: %v", peerPublicKey, err)
		}
		delete(nm.peerConnections, peerPublicKey)
	}

	// Set state to disconnected
	nm.connectionState = ConnectionStateDisconnected
	nm.RealtimeData.SetConnectionState(data.StateDisconnected)
	nm.RealtimeData.SetStatusMessage("Disconnected")
	nm.RealtimeData.SetComputerIP("0.0.0.0") // Clear the IP when disconnected
	nm.ReconnectAttempts = 0
	nm.RealtimeData.SetNetworks([]smodels.ComputerNetworkInfo{}) // Clear the network list

	// Update UI
	nm.refreshUI()
	nm.refreshNetworkList() // Added this line

	return nil
}




