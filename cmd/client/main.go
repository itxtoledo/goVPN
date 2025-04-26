package main

import (
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Configurar o log para facilitar o debug
	logFile, _ := os.OpenFile("govpn.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if logFile != nil {
		// log.SetOutput(logFile)
	}

	// Inicialização do caminho de dados
	setupDataPath()

	// Inicializar UIManager que contém a camada de dados em tempo real
	ui := NewUIManager()

	// Executar a aplicação
	ui.Run()
}

// setupDataPath cria os diretórios necessários para o aplicativo
func setupDataPath() {
	// Obter o caminho do diretório de dados
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}

	// Criar o diretório de dados se não existir
	dataPath := filepath.Join(homeDir, ".govpn")
	err = os.MkdirAll(dataPath, 0755)
	if err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
}
