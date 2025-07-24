package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/storage"
	sclient "github.com/itxtoledo/govpn/libs/signaling/client"
	smodels "github.com/itxtoledo/govpn/libs/signaling/models"
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
type NetworkManager struct {
	VirtualNetwork    NetworkInterface
	SignalingServer   *sclient.SignalingClient
	NetworkID         string
	connectionState   ConnectionState
	ReconnectAttempts int
	MaxReconnects     int

	// Dependencies
	RealtimeData       *data.RealtimeDataLayer
	ConfigManager      *storage.ConfigManager
	refreshNetworkList func()
	refreshUI          func()
}

// NewNetworkManager creates a new instance of NetworkManager
func NewNetworkManager(realtimeData *data.RealtimeDataLayer, configManager *storage.ConfigManager, refreshNetworkList func(), refreshUI func()) *NetworkManager {
	nm := &NetworkManager{
		connectionState:    ConnectionStateDisconnected,
		ReconnectAttempts:  0,
		MaxReconnects:      5,
		RealtimeData:       realtimeData,
		ConfigManager:      configManager,
		refreshNetworkList: refreshNetworkList,
		refreshUI:          refreshUI,
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
			nm.refreshNetworkList()
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

// Disconnect disconnects from the VPN network
func (nm *NetworkManager) Disconnect() error {
	if nm.connectionState == ConnectionStateDisconnected {
		return nil
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

	// Set state to disconnected
	nm.connectionState = ConnectionStateDisconnected
	nm.RealtimeData.SetConnectionState(data.StateDisconnected)
	nm.RealtimeData.SetStatusMessage("Disconnected")
	nm.RealtimeData.SetComputerIP("0.0.0.0") // Clear the IP when disconnected
	nm.ReconnectAttempts = 0

	// Update UI
	nm.refreshUI()

	return nil
}
