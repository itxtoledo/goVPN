package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"github.com/itxtoledo/govpn/libs/crypto_utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pion/webrtc/v3"
)

// VPNClient é a estrutura principal do cliente VPN
type VPNClient struct {
	DB             *sql.DB
	CurrentRoom    string
	IsConnected    bool
	NetworkManager *NetworkManager
	UI             *UIManager

	// Chaves RSA para identificação do usuário
	PrivateKeyPEM string // Chave privada em formato PEM
	PublicKeyPEM  string // Chave pública em formato PEM - Agora é o identificador principal
}

// NewVPNClient cria uma nova instância do cliente VPN
func NewVPNClient() *VPNClient {
	vpn := &VPNClient{
		IsConnected: false,
	}

	// Inicialização do banco de dados
	if err := vpn.initDatabase(); err != nil {
		log.Fatalf("Erro ao inicializar banco de dados: %v", err)
	}

	// Carrega ou gera chaves RSA para identificação do usuário
	// Esta etapa é essencial e deve acontecer logo na inicialização
	if err := vpn.loadOrGenerateKeys(); err != nil {
		log.Fatalf("Erro crítico ao carregar/gerar chaves RSA: %v", err)
		// Se não conseguirmos gerar as chaves, não faz sentido continuar
	}

	log.Printf("Chaves RSA carregadas com sucesso: chave pública disponível")

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
	var signalServer string
	err := v.DB.QueryRow("SELECT value FROM settings WHERE key = 'signal_server'").Scan(&signalServer)
	if err == nil && signalServer != "" {
		v.NetworkManager.SignalServer = signalServer
	}

	// Carrega as configurações de STUN
	var stunServer string
	err = v.DB.QueryRow("SELECT value FROM settings WHERE key = 'stun_server'").Scan(&stunServer)
	if err == nil && stunServer != "" {
		v.NetworkManager.ICEServers = []webrtc.ICEServer{
			{
				URLs: []string{stunServer},
			},
		}
	}

	// Outras configurações podem ser carregadas aqui
}

// Run inicia o cliente VPN
func (v *VPNClient) Run() {
	log.Println("Iniciando cliente goVPN")

	// Inicia a interface gráfica
	v.UI.Start()
}

// initDatabase inicializa o banco de dados SQLite
func (v *VPNClient) initDatabase() error {
	// Cria o diretório de dados do usuário se não existir
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	appDir := filepath.Join(userConfigDir, "goVPN")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(appDir, "govpn.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Cria tabelas se não existirem
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS rooms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			password TEXT NOT NULL,
			last_connected TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
			);
		
		CREATE TABLE IF NOT EXISTS keys (
			id TEXT PRIMARY KEY,
			private_key TEXT NOT NULL,
			public_key TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	v.DB = db
	return nil
}

// loadOrGenerateKeys carrega as chaves RSA do banco ou gera novas se não existirem
func (v *VPNClient) loadOrGenerateKeys() error {
	// Tenta carregar as chaves existentes primeiro
	var privateKey, publicKey string
	err := v.DB.QueryRow("SELECT private_key, public_key FROM keys WHERE id = 'user_key' LIMIT 1").Scan(&privateKey, &publicKey)

	if err == nil {
		// Chaves encontradas, carrega-as
		v.PrivateKeyPEM = privateKey
		v.PublicKeyPEM = publicKey
		log.Println("Chaves RSA carregadas com sucesso.")
		return nil
	}

	if err != sql.ErrNoRows {
		// Erro diferente de "não encontrado"
		return err
	}

	// Chaves não encontradas, gera novas
	log.Println("Gerando novas chaves RSA...")
	privateKey, publicKey, err = crypto_utils.GenerateRSAKeys()
	if err != nil {
		return err
	}

	// Armazena as chaves no banco de dados
	_, err = v.DB.Exec("INSERT INTO keys (id, private_key, public_key) VALUES ('user_key', ?, ?)",
		privateKey, publicKey)
	if err != nil {
		return err
	}

	// Atualiza a estrutura VPNClient
	v.PrivateKeyPEM = privateKey
	v.PublicKeyPEM = publicKey
	log.Println("Novas chaves RSA geradas e armazenadas com sucesso.")

	return nil
}

// getPublicKey retorna a chave pública, garantindo que ela esteja carregada
func (v *VPNClient) getPublicKey() string {
	// Se a chave pública não estiver carregada, tenta carregá-la
	if v.PublicKeyPEM == "" {
		err := v.loadOrGenerateKeys()
		if err != nil {
			log.Printf("Erro ao carregar/gerar chaves RSA: %v", err)
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
