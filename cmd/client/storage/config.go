package storage

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Config representa as configurações da aplicação
type Config struct {
	Username      string `json:"username"`
	ServerAddress string `json:"server_address"`
	Theme         string `json:"theme"`
	Language      string `json:"language"`
}

// ConfigManager gerencia as configurações da aplicação
type ConfigManager struct {
	config     Config
	configPath string
	mutex      sync.Mutex
}

// NewConfigManager cria uma nova instância do gerenciador de configurações
func NewConfigManager() *ConfigManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting user home directory: %v", err)
		homeDir = "."
	}

	dataPath := filepath.Join(homeDir, ".govpn")
	configPath := filepath.Join(dataPath, "config.json")

	cm := &ConfigManager{
		configPath: configPath,
		config: Config{
			Username:      "User",
			ServerAddress: "wss://echo.websocket.org",
			Theme:         "system",
			Language:      "en",
		},
	}

	// Cria o diretório de dados se não existir
	err = os.MkdirAll(dataPath, 0755)
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

// UpdateUsername atualiza o nome de usuário
func (cm *ConfigManager) UpdateUsername(username string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.config.Username = username
	return cm.SaveConfig()
}

// UpdateServerAddress atualiza o endereço do servidor
func (cm *ConfigManager) UpdateServerAddress(address string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.config.ServerAddress = address
	return cm.SaveConfig()
}

// UpdateTheme atualiza o tema
func (cm *ConfigManager) UpdateTheme(theme string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.config.Theme = theme
	return cm.SaveConfig()
}

// UpdateLanguage atualiza o idioma
func (cm *ConfigManager) UpdateLanguage(language string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.config.Language = language
	return cm.SaveConfig()
}

// LoadConfig carrega as configurações do arquivo
func (cm *ConfigManager) LoadConfig() {
	file, err := os.Open(cm.configPath)
	if err != nil {
		// Se o arquivo não existe, cria com valores padrão
		if os.IsNotExist(err) {
			cm.SaveConfig()
		}
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cm.config)
	if err != nil {
		log.Printf("Error decoding config file: %v", err)
	}
}

// SaveConfig salva as configurações no arquivo
func (cm *ConfigManager) SaveConfig() error {
	file, err := os.Create(cm.configPath)
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
