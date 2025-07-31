package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Config representa as configurações da aplicação
type Config struct {
	ComputerName  string `json:"computername"`
	ServerAddress string `json:"server_address"`
	Language      string `json:"language"`
	PublicKey     string `json:"public_key"`
	PrivateKey    string `json:"private_key"`
}

// Network represents a VPN network

// ConfigManager gerencia as configurações da aplicação
type ConfigManager struct {
	config   Config
	dataPath string // Add dataPath field
	mutex    sync.Mutex
}

// NewConfigManager cria uma nova instância do gerenciador de configurações
func NewConfigManager(customConfigPath string) *ConfigManager {
	var dataPath string

	if customConfigPath != "" {
		dataPath = customConfigPath
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Error getting user home directory: %v", err)
			homeDir = "."
		}
		
		// Use appropriate config directory for each OS:
		// - Windows: %LOCALAPPDATA%/govpn
		// - macOS: ~/Library/Application Support/govpn
		// - Linux: ~/.local/share/govpn
		switch runtime.GOOS {
		case "windows":
			dataPath = filepath.Join(os.Getenv("LOCALAPPDATA"), "govpn")
		case "darwin": // macOS
			dataPath = filepath.Join(homeDir, "Library", "Application Support", "govpn")
		default: // Linux and others
			dataPath = filepath.Join(homeDir, ".local", "share", "govpn")
		}
	}

	cm := &ConfigManager{
		dataPath: dataPath, // Initialize dataPath
		config: Config{
			ComputerName: "Computer",
			// ServerAddress: "wss://govpn-k6ql.onrender.com:8080/ws",
			ServerAddress: "wss://localhost:8080/ws",
			Language:      "en",
		},
	}

	// Cria o diretório de dados se não existir
	err := os.MkdirAll(dataPath, 0755)
	if err != nil {
		log.Printf("Error creating data directory: %v", err)
	}

	// Carrega as configurações do arquivo
	cm.LoadConfig()

	return cm
}

// GetConfig retorna as configurações atuais
func (cm *ConfigManager) GetConfig() Config {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	return cm.config
}

// UpdateConfig atualiza as configurações
func (cm *ConfigManager) UpdateConfig(config Config) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.config = config
	return cm.SaveConfig()
}

// UpdateComputerName atualiza o nome de usuário
func (cm *ConfigManager) UpdateComputerName(computername string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.config.ComputerName = computername
	return cm.SaveConfig()
}

// UpdateServerAddress atualiza o endereço do servidor
func (cm *ConfigManager) UpdateServerAddress(address string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.config.ServerAddress = address
	return cm.SaveConfig()
}



// UpdateLanguage atualiza o idioma
func (cm *ConfigManager) UpdateLanguage(language string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.config.Language = language
	return cm.SaveConfig()
}

// GetKeyPair retorna as chaves pública e privada
func (cm *ConfigManager) GetKeyPair() (string, string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	return cm.config.PublicKey, cm.config.PrivateKey
}

// LoadConfig carrega as configurações do arquivo
func (cm *ConfigManager) LoadConfig() {
	configPath := filepath.Join(cm.dataPath, "config.json")
	log.Printf("Loading config from: %s", configPath)

	file, err := os.Open(configPath)
	if err != nil {
		// Se o arquivo não existe, cria com valores padrão
		if os.IsNotExist(err) {
			log.Printf("Config file doesn't exist, creating with default values")
			cm.SaveConfig()
		} else {
			log.Printf("Error opening config file: %v", err)
		}
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cm.config)
	if err != nil {
		log.Printf("Error decoding config file: %v", err)
		return
	}

	// Log config details
	log.Printf("Config loaded successfully - ComputerName: %s, Language: %s",
		cm.config.ComputerName, cm.config.Language)

	// Check for public/private keys
	if cm.config.PublicKey != "" && cm.config.PrivateKey != "" {
		log.Printf("Key pair found in config - Public key prefix: %s...", cm.config.PublicKey[:10])
	} else {
		log.Printf("WARNING: No key pair found in config file - Public key empty: %v, Private key empty: %v",
			cm.config.PublicKey == "", cm.config.PrivateKey == "")

		log.Println("Generating new Ed25519 key pair")
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)

		if err != nil {
			log.Printf("Error generating key pair: %v", err)
		}

		// Convert keys to string for storage
		publicKeyStr := base64.StdEncoding.EncodeToString(publicKey)
		privateKeyStr := base64.StdEncoding.EncodeToString(privateKey)

		log.Printf("Generated new public key: %s...", publicKeyStr[:10])
		log.Printf("Generated new private key: %s...", privateKeyStr[:10])

		// Update config with new keys
		cm.config.PublicKey = publicKeyStr
		cm.config.PrivateKey = privateKeyStr

		cm.SaveConfig()
	}
}

// GetDataPath returns the data directory path
func (cm *ConfigManager) GetDataPath() string {
	return cm.dataPath
}

// SaveConfig salva as configurações no arquivo
func (cm *ConfigManager) SaveConfig() error {
	configPath := filepath.Join(cm.dataPath, "config.json")
	file, err := os.Create(configPath)
	if err != nil {
		log.Printf("Error creating config file: %v", err)
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(cm.config)
	if err != nil {
		log.Printf("Error encoding config file: %v", err)
		return err
	}

	return nil
}
