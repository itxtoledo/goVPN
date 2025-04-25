package storage

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// Config represents application settings
type Config struct {
	SignalServer    string `json:"signal_server"`
	ThemePreference string `json:"theme_preference"`
	Username        string `json:"username"` // Added username field
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *Config {
	return &Config{
		SignalServer:    "localhost:8080",
		ThemePreference: "System Default",
		Username:        "",
	}
}

// ConfigManager manages application configuration
type ConfigManager struct {
	configPath string
	config     *Config
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("Failed to get user config directory: %v", err)
		configDir = "."
	}

	// Create goVPN directory if it doesn't exist
	goVPNDir := filepath.Join(configDir, "goVPN")
	if err := os.MkdirAll(goVPNDir, 0755); err != nil {
		log.Printf("Failed to create config directory: %v", err)
	}

	configPath := filepath.Join(goVPNDir, "config.json")

	cm := &ConfigManager{
		configPath: configPath,
		config:     DefaultConfig(),
	}

	// Try to load existing configuration
	cm.LoadConfig()

	return cm
}

// LoadConfig loads configuration from file
func (cm *ConfigManager) LoadConfig() {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read config file: %v", err)
		}
		return
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Failed to parse config file: %v", err)
		return
	}

	cm.config = &config
}

// SaveConfig saves configuration to file
func (cm *ConfigManager) SaveConfig() {
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal config: %v", err)
		return
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		log.Printf("Failed to write config file: %v", err)
	}
}

// GetConfig returns current configuration
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// UpdateConfig updates configuration
func (cm *ConfigManager) UpdateConfig(config *Config) {
	cm.config = config
	cm.SaveConfig()
}
