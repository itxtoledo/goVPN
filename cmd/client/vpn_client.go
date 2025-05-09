package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// VPNClient é a estrutura principal do cliente VPN
type VPNClient struct {
	PrivateKey     interface{}
	PublicKey      interface{}
	PublicKeyStr   string // Identificador principal do cliente
	Computers      []Computer
	CurrentRoom    string
	IsConnected    bool
	NetworkManager *NetworkManager
	UI             *UIManager
	ConfigManager  *ConfigManager
}

// NewVPNClient creates a new VPN client
func NewVPNClient(ui *UIManager) *VPNClient {
	// Generate an Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Printf("Error generating key pair: %v", err)
	}

	// Convert public key to string for storage
	publicKeyStr := base64.StdEncoding.EncodeToString(publicKey)

	// Create VPN client
	client := &VPNClient{
		UI:            ui,
		IsConnected:   false,
		PrivateKey:    privateKey,
		PublicKey:     publicKey,
		PublicKeyStr:  publicKeyStr,
		Computers:     make([]Computer, 0),
		ConfigManager: ui.ConfigManager, // Set ConfigManager from UI manager
	}

	// Initialize rooms as an empty slice (in-memory storage only)
	ui.Rooms = make([]*storage.Room, 0)

	// Create network manager
	client.NetworkManager = NewNetworkManager(ui)

	return client
}

// loadSettings carrega as configurações salvas do banco de dados
func (v *VPNClient) loadSettings() {
	// Carrega as configurações de usuário do config manager
	config := v.ConfigManager.GetConfig()

	// Atualiza o nome de usuário na camada de dados em tempo real
	v.UI.RealtimeData.SetUsername(config.Username)

	// Atualiza o endereço do servidor na camada de dados em tempo real
	v.UI.RealtimeData.SetServerAddress(config.ServerAddress)

	// Aplicar o tema
	switch config.Theme {
	case "light":
		v.UI.App.Settings().SetTheme(fyne.Theme(theme.LightTheme()))
	case "dark":
		v.UI.App.Settings().SetTheme(fyne.Theme(theme.DarkTheme()))
	default:
		// Tema do sistema é o padrão, não é necessário configurá-lo explicitamente
	}

	// Configura o idioma (se implementado)
	if config.Language != "" {
		v.UI.RealtimeData.SetLanguage(config.Language)
	}

	log.Printf("Settings loaded: Username=%s, Theme=%s, Language=%s, Server=%s",
		config.Username, config.Theme, config.Language, config.ServerAddress)
}

// Run inicia o cliente VPN
func (v *VPNClient) Run() {
	log.Println("Starting goVPN client")

	// Attempt to connect to the backend in a background goroutine
	go func() {
		// Defina o estado inicial na camada de dados
		v.UI.RealtimeData.SetConnectionState(data.StateDisconnected)
		v.UI.RealtimeData.SetStatusMessage("Starting...")

		// Obter o endereço do servidor das configurações
		config := v.ConfigManager.GetConfig()
		serverAddress := config.ServerAddress

		// Usar endereço padrão se não estiver definido
		if serverAddress == "" {
			serverAddress = "ws://localhost:8080"
			log.Println("No server address configured, using default:", serverAddress)
		}

		// Inicializar o signaling client no NetworkManager se necessário
		if v.NetworkManager.SignalingServer == nil {
			v.NetworkManager.SignalingServer = NewSignalingClient(v.UI, v.PublicKeyStr)
		}

		// Set the reference to this VPNClient in the SignalingClient
		v.NetworkManager.SignalingServer.SetVPNClient(v)

		// Tentativa de conexão ao servidor de backend
		log.Printf("Iniciando conexão automática com o servidor de sinalização")
		log.Printf("Attempting to connect to backend server: %s", serverAddress)
		v.UI.RealtimeData.SetStatusMessage("Connecting to backend...")

		// Conectar ao servidor
		err := v.NetworkManager.Connect(serverAddress)

		if err != nil {
			log.Printf("Background connection attempt failed: %v", err)
			v.UI.RealtimeData.SetStatusMessage("Connection failed")
			v.UI.RealtimeData.EmitEvent(data.EventError, fmt.Sprintf("Connection failed: %v", err), nil)
		} else {
			log.Println("Successfully connected to backend server in background")
			v.UI.RealtimeData.SetStatusMessage("Connected")

			// Atualizar a lista de salas
			v.UI.refreshNetworkList()
		}
	}()
}

// generateClientID gera um ID de cliente único
func generateClientID() string {
	// Geramos um identificador simples baseado em timestamp + random
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())

	// Adicionar 8 bytes aleatórios
	b := make([]byte, 8)
	rand.Read(b)
	randomHex := hex.EncodeToString(b)

	return fmt.Sprintf("%s-%s", timestamp, randomHex)
}

// GetIdentifier retorna o identificador do cliente
func (v *VPNClient) GetIdentifier() string {
	return v.PublicKeyStr
}

// GetConfig obtém uma configuração do ambiente ou retorna um valor padrão
func (c *VPNClient) GetConfig(key string) string {
	value := os.Getenv(key)
	return value
}
