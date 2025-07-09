package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/itxtoledo/govpn/cmd/client/data"
)

// VPNClient é a estrutura principal do cliente VPN
type VPNClient struct {
	PrivateKey     interface{}
	PublicKey      interface{}
	PublicKeyStr   string // Identificador principal do cliente
	ComputerName   string
	Computers      []Computer
	CurrentNetwork string
	IsConnected    bool
	NetworkManager *NetworkManager
	ConfigManager  *ConfigManager
}

// NewVPNClient creates a new VPN client
func NewVPNClient(configManager *ConfigManager, defaultWebsocketURL string, computername string) *VPNClient {
	var privateKey ed25519.PrivateKey
	var publicKey ed25519.PublicKey
	var publicKeyStr string

	log.Println("Initializing VPN client...")

	// Load existing keys from config
	publicKeyStr, privateKeyStr := configManager.GetKeyPair()

	log.Printf("Loaded public key from config: %s...", publicKeyStr[:10])
	log.Printf("Loaded private key from config: %s...", privateKeyStr[:10])

	// Decode public key from base64
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil {
		log.Printf("Error decoding public key, generating new one: %v", err)
	} else {
		log.Printf("Successfully decoded public key, length: %d bytes", len(publicKeyBytes))
		publicKey = ed25519.PublicKey(publicKeyBytes)
	}

	// Decode private key from base64
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyStr)
	if err != nil {
		log.Printf("Error decoding private key, generating new one: %v", err)
	} else {
		log.Printf("Successfully decoded private key, length: %d bytes", len(privateKeyBytes))
		privateKey = ed25519.PrivateKey(privateKeyBytes)
	}

	// Create VPN client
	client := &VPNClient{
		IsConnected:   false,
		PrivateKey:    privateKey,
		PublicKey:     publicKey,
		PublicKeyStr:  publicKeyStr,
		ComputerName:  computername,
		Computers:     make([]Computer, 0),
		ConfigManager: configManager,
	}

	// TODO client.PublicKeyStr esta vazio
	// Check if PublicKeyStr has content
	if client.PublicKeyStr == "" {
		log.Println("WARNING: PublicKeyStr is empty!")
	} else {
		log.Printf("PublicKeyStr has content with length %d", len(client.PublicKeyStr))
	}

	log.Printf("VPN client initialized with public key: %s...", client.PublicKeyStr)

	return client
}

// SetupNetworkManager creates and configures the NetworkManager for the VPN client
func (v *VPNClient) SetupNetworkManager(realtimeData *data.RealtimeDataLayer, refreshNetworkList func(), refreshUI func()) {
	v.NetworkManager = NewNetworkManager(realtimeData, v.ConfigManager, refreshNetworkList, refreshUI)
}

// loadSettings carrega as configurações salvas do banco de dados
func (v *VPNClient) loadSettings(realtimeData *data.RealtimeDataLayer, app fyne.App) {
	// Carrega as configurações de usuário do config manager
	config := v.ConfigManager.GetConfig()

	// Atualiza o nome de usuário na camada de dados em tempo real
	realtimeData.SetComputerName(config.ComputerName)

	// Atualiza o endereço do servidor na camada de dados em tempo real
	realtimeData.SetServerAddress(config.ServerAddress)

	// Aplicar o tema
	switch config.Theme {
	case "light":
		lightTheme := theme.LightTheme()
		customTheme := NewCustomTheme(lightTheme)
		app.Settings().SetTheme(customTheme)
	case "dark":
		darkTheme := theme.DarkTheme()
		customTheme := NewCustomTheme(darkTheme)
		app.Settings().SetTheme(customTheme)
	default:
		// Tema do sistema com melhorias de contraste
		currentTheme := app.Settings().Theme()
		customTheme := NewCustomTheme(currentTheme)
		app.Settings().SetTheme(customTheme)
	}

	// Configura o idioma (se implementado)
	if config.Language != "" {
		realtimeData.SetLanguage(config.Language)
	}

	log.Printf("Settings loaded: ComputerName=%s, Theme=%s, Language=%s, Server=%s",
		config.ComputerName, config.Theme, config.Language, config.ServerAddress)
}

// Run inicia o cliente VPN
func (v *VPNClient) Run(defaultWebsocketURL string, realtimeData *data.RealtimeDataLayer, refreshNetworkList func(), refreshUI func()) {
	log.Println("Starting goVPN client")

	// Setup the network manager first
	if v.NetworkManager == nil {
		v.SetupNetworkManager(realtimeData, refreshNetworkList, refreshUI)
	}

	// Attempt to connect to the backend in a background goroutine
	go func() {
		// Defina o estado inicial na camada de dados
		realtimeData.SetConnectionState(data.StateDisconnected)
		realtimeData.SetStatusMessage("Starting...")

		// Obter o endereço do servidor das configurações
		config := v.ConfigManager.GetConfig()
		serverAddress := config.ServerAddress

		// Usar endereço padrão se não estiver definido
		if serverAddress == "" {
			serverAddress = defaultWebsocketURL
			log.Println("No server address configured, using default from build:", serverAddress)
		}

		// Tentativa de conexão ao servidor de backend
		log.Printf("Iniciando conexão automática com o servidor de sinalização")
		log.Printf("Attempting to connect to backend server: %s", serverAddress)
		realtimeData.SetStatusMessage("Connecting to backend...")

		// Conectar ao servidor
		err := v.NetworkManager.Connect(serverAddress)

		if err != nil {
			log.Printf("Background connection attempt failed: %v", err)
			realtimeData.SetStatusMessage("Connection failed")
			realtimeData.EmitEvent(data.EventError, fmt.Sprintf("Connection failed: %v", err), nil)
		} else {
			log.Println("Successfully connected to backend server in background")
			realtimeData.SetStatusMessage("Connected")

			// Atualizar a lista de salas
			refreshNetworkList()
		}
	}()
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
