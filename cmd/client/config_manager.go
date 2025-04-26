package main

import (
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// ConfigManager gerencia as configurações da aplicação
type ConfigManager struct {
	*storage.ConfigManager
}

// NewConfigManager cria uma nova instância do gerenciador de configurações
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		ConfigManager: storage.NewConfigManager(),
	}
}

// Constantes da aplicação
const (
	AppTitleName   = "GoVPN"
	AppVersion     = "0.1.0"
	AppDescription = "VPN P2P para redes locais e internet"
	AppAuthor      = "Gustavo Toledo"
	AppRepository  = "https://github.com/itxtoledo/govpn"
	AppLicense     = "MIT License"
)
