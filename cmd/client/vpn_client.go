package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/itxtoledo/govpn/cmd/client/storage"
	"github.com/itxtoledo/govpn/libs/crypto_utils"
	_ "github.com/mattn/go-sqlite3"
)

// VPNClient é a estrutura principal do cliente VPN
type VPNClient struct {
	DBManager      *storage.DatabaseManager
	CurrentRoom    string
	IsConnected    bool
	NetworkManager *NetworkManager
	UI             *UIManager
	Config         *storage.ConfigManager

	// Chaves RSA para identificação do usuário
	PrivateKeyPEM string // Chave privada em formato PEM
	PublicKeyPEM  string // Chave pública em formato PEM - Agora é o identificador principal
}

// NewVPNClient cria uma nova instância do cliente VPN
func NewVPNClient() *VPNClient {
	vpn := &VPNClient{
		IsConnected: false,
		Config:      storage.NewConfigManager(),
	}

	// Inicialização do banco de dados
	dbManager, err := storage.NewDatabaseManager()
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	vpn.DBManager = dbManager

	// Carrega ou gera chaves RSA para identificação do usuário
	// Esta etapa é essencial e deve acontecer logo na inicialização
	if err := vpn.loadOrGenerateKeys(); err != nil {
		log.Fatalf("Critical error loading/generating RSA keys: %v", err)
		// Se não conseguirmos gerar as chaves, não faz sentido continuar
	}

	log.Printf("RSA keys successfully loaded: public key available")

	// Inicialização do gerenciador de rede
	vpn.NetworkManager = NewNetworkManager(vpn)

	// Carrega configurações do banco de dados
	vpn.loadSettings()

	// Inicialização da interface gráfica (separada da inicialização completa)
	// Isso evita dependências circulares durante a inicialização
	vpn.UI = NewUIManager(vpn)

	return vpn
}

// loadSettings carrega as configurações salvas do banco de dados
func (v *VPNClient) loadSettings() {
	// Carrega as configurações de sinalização
	signalServer, err := v.DBManager.LoadSignalServer()
	if err == nil && signalServer != "" {
		v.NetworkManager.SignalServer = signalServer
	}

	// Carrega as configurações de STUN/ICE servers
	iceServers, err := v.DBManager.GetICEServers()
	if err == nil && len(iceServers) > 0 {
		v.NetworkManager.RTCConfig.ICEServers = iceServers
	}

	// Outras configurações podem ser carregadas aqui
}

// Run inicia o cliente VPN
func (v *VPNClient) Run() {
	log.Println("Starting goVPN client")

	// Attempt to connect to the backend in a background goroutine
	go func() {
		log.Println("Attempting to connect to backend server...")
		err := v.NetworkManager.Connect()
		if err != nil {
			log.Printf("Background connection attempt failed: %v", err)
		} else {
			log.Println("Successfully connected to backend server in background")
			v.NetworkManager.GetRoomList() // Fetch the room list while we're at it

			// Update the UI power button state after successful connection
			// This is safe because the Fyne UI uses a queue system that handles UI updates from goroutines
			if v.UI != nil {
				// A small delay to ensure UI is fully initialized
				// This is important as UI might not be ready immediately after app start
				time.Sleep(500 * time.Millisecond)
				v.UI.refreshNetworkList()
			}
		}
	}()

	// Inicia a interface gráfica
	v.UI.Start()
}

// loadOrGenerateKeys carrega as chaves RSA do banco ou gera novas se não existirem
func (v *VPNClient) loadOrGenerateKeys() error {
	// Tenta carregar as chaves existentes primeiro
	privateKey, publicKey, err := v.DBManager.LoadRSAKeys()

	if err == nil {
		// Chaves encontradas, carrega-as
		v.PrivateKeyPEM = privateKey
		v.PublicKeyPEM = publicKey
		log.Println("RSA keys successfully loaded.")
		return nil
	}

	if err != sql.ErrNoRows {
		// Erro diferente de "não encontrado"
		return err
	}

	// Chaves não encontradas, gera novas
	log.Println("Generating new RSA keys...")
	privateKey, publicKey, err = crypto_utils.GenerateRSAKeys()
	if err != nil {
		return err
	}

	// Armazena as chaves no banco de dados
	err = v.DBManager.SaveRSAKeys(privateKey, publicKey)
	if err != nil {
		return err
	}

	// Atualiza a estrutura VPNClient
	v.PrivateKeyPEM = privateKey
	v.PublicKeyPEM = publicKey
	log.Println("New RSA keys generated and stored successfully.")

	return nil
}

// getPublicKey retorna a chave pública, garantindo que ela esteja carregada
func (v *VPNClient) getPublicKey() string {
	// Se a chave pública não estiver carregada, tenta carregá-la
	if v.PublicKeyPEM == "" {
		err := v.loadOrGenerateKeys()
		if err != nil {
			log.Printf("Error loading/generating RSA keys: %v", err)
			return ""
		}
	}

	return v.PublicKeyPEM
}

// GetConfig obtém uma configuração do ambiente ou retorna um valor padrão
func (c *VPNClient) GetConfig(key string) string {
	value := os.Getenv(key)
	return value
}
