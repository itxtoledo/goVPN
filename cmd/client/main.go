package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/icon"
)

var websocketURL = "wss://govpn-k6ql.onrender.com/ws"

func main() {
	// Configurar o log para facilitar o debug
	logFile, _ := os.OpenFile("govpn.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if logFile != nil {
		// log.SetOutput(logFile)
	}

	// Inicialização do caminho de dados
	setupDataPath()

	// Inicializar UIManager que contém a camada de dados em tempo real
	ui := NewUIManager(websocketURL)

	// Set up system tray
	if desk, ok := ui.App.(desktop.App); ok {
		desk.SetSystemTrayIcon(fyne.NewStaticResource("appIcon", icon.AppIcon.Content()))

		// Create menu items
		showItem := fyne.NewMenuItem("Show", func() {
			ui.MainWindow.Show()
		})
		quitItem := fyne.NewMenuItem("Quit", func() {
			ui.App.Quit()
		})

		connectItem := fyne.NewMenuItem("Connect", func() {
			if ui.VPN != nil {
				// The Run method handles the connection logic
				go ui.VPN.Run(websocketURL)
			}
		})

		disconnectItem := fyne.NewMenuItem("Disconnect", func() {
			if ui.VPN != nil && ui.VPN.NetworkManager != nil {
				go ui.VPN.NetworkManager.Disconnect()
			}
		})

		// Create the menu with separators for better organization
		menu := fyne.NewMenu("GoVPN", showItem, fyne.NewMenuItemSeparator(), connectItem, disconnectItem, fyne.NewMenuItemSeparator(), quitItem)
		desk.SetSystemTrayMenu(menu)

		// Add a listener to update the menu based on connection state
		ui.RealtimeData.ConnectionState.AddListener(binding.NewDataListener(func() {
			state, _ := ui.RealtimeData.ConnectionState.Get()
			connectionState := data.ConnectionState(state)

			connectItem.Disabled = (connectionState != data.StateDisconnected)
			disconnectItem.Disabled = (connectionState == data.StateDisconnected)

			// Update the menu to reflect the new state
			desk.SetSystemTrayMenu(menu)
		}))

		// Set initial state (app starts disconnected)
		connectItem.Disabled = false
		disconnectItem.Disabled = true
		desk.SetSystemTrayMenu(menu)
	}

	// Hide window on close
	ui.MainWindow.SetCloseIntercept(func() {
		ui.MainWindow.Hide()
	})

	// Executar a aplicação
	ui.Run(websocketURL)
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
