package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/storage"
	_ "github.com/mattn/go-sqlite3"
)

// VPNClient é a estrutura principal do cliente VPN
type VPNClient struct {
	DBManager      *storage.DatabaseManager
	CurrentRoom    string
	IsConnected    bool
	NetworkManager *NetworkManager
	UI             *UIManager
	ConfigManager  *ConfigManager

	// Identificação do cliente
	PublicKeyStr string // Identificador principal do cliente
}

// NewVPNClient cria uma nova instância do cliente VPN
func NewVPNClient(ui *UIManager) *VPNClient {
	vpn := &VPNClient{
		IsConnected:   false,
		UI:            ui,
		ConfigManager: ui.ConfigManager,
	}

	// Inicialização do banco de dados
	dbManager, err := storage.NewDatabaseManager()
	if err != nil {
		log.Printf("Error initializing database: %v", err)
	} else {
		vpn.DBManager = dbManager

		// Carrega ou gera identificador do cliente
		if err := vpn.loadOrGenerateIdentifier(); err != nil {
			log.Printf("Critical error loading/generating client identifier: %v", err)
			// Se não conseguirmos gerar o identificador, não faz sentido continuar
		} else {
			log.Printf("Client identifier successfully loaded")
		}
	}

	// Inicialização do gerenciador de rede
	vpn.NetworkManager = NewNetworkManager(ui)

	// Carrega configurações
	vpn.loadSettings()

	return vpn
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

// loadOrGenerateIdentifier carrega o identificador do cliente do banco ou gera um novo se não existir
func (v *VPNClient) loadOrGenerateIdentifier() error {
	// Tenta carregar o identificador existente primeiro
	_, publicKey, err := v.DBManager.LoadKeys()

	if err == nil && publicKey != "" {
		// Identificador encontrado
		v.PublicKeyStr = publicKey
		log.Println("Client identifier successfully loaded.")
		return nil
	} else if err != sql.ErrNoRows && err != nil {
		// Erro diferente de "não encontrado"
		return err
	}

	// Identificador não encontrado, gera novo
	log.Println("No client identifier found, generating new one...")

	// Gerar id simples (pode ser um UUID ou outro identificador único)
	publicKey = generateClientID()

	// Armazena o identificador no banco de dados
	err = v.DBManager.SaveKeys("", publicKey)
	if err != nil {
		return err
	}

	// Atualiza a estrutura VPNClient
	v.PublicKeyStr = publicKey
	log.Println("New client identifier generated and stored successfully.")

	return nil
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
	// Se o identificador não estiver carregado, tenta carregá-lo
	if v.PublicKeyStr == "" {
		err := v.loadOrGenerateIdentifier()
		if err != nil {
			log.Printf("Error loading/generating client identifier: %v", err)
			return ""
		}
	}

	return v.PublicKeyStr
}

// GetConfig obtém uma configuração do ambiente ou retorna um valor padrão
func (c *VPNClient) GetConfig(key string) string {
	value := os.Getenv(key)
	return value
}
