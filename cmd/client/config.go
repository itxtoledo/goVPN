package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// UserConfig stores user configuration
type UserConfig struct {
	SignalServer        string `json:"signalServer"`
	ThemePreference     string `json:"themePreference"`     // "System Default", "Light", "Dark"
	Language            string `json:"language"`            // "English", "Spanish"
	AutoConnect         bool   `json:"autoConnect"`         // Auto-connect to last network
	StartOnSystemBoot   bool   `json:"startOnSystemBoot"`   // Start on system startup
	EnableNotifications bool   `json:"enableNotifications"` // Enable notifications
	LastConnectedRoom   string `json:"lastConnectedRoom"`   // Last room the user connected to
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *UserConfig {
	return &UserConfig{
		SignalServer:        "localhost:8080",
		ThemePreference:     "System Default",
		Language:            "English",
		AutoConnect:         false,
		StartOnSystemBoot:   false,
		EnableNotifications: true,
		LastConnectedRoom:   "",
	}
}

// ConfigManager manages application configuration
type ConfigManager struct {
	configPath string
	config     *UserConfig
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
	data, err := ioutil.ReadFile(cm.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read config file: %v", err)
		}
		return
	}

	var config UserConfig
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

	if err := ioutil.WriteFile(cm.configPath, data, 0644); err != nil {
		log.Printf("Failed to write config file: %v", err)
	}
}

// GetConfig returns current configuration
func (cm *ConfigManager) GetConfig() *UserConfig {
	return cm.config
}

// UpdateConfig updates configuration
func (cm *ConfigManager) UpdateConfig(config *UserConfig) {
	cm.config = config
	cm.SaveConfig()
}
