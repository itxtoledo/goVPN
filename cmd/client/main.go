package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/icon"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

var websocketURL = "wss://govpn-k6ql.onrender.com/ws"

func main() {
	logFile, _ := os.OpenFile("govpn.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if logFile != nil {
		// log.SetOutput(logFile)
	}

	setupDataPath()

	configManager := storage.NewConfigManager()
	username := configManager.GetConfig().Username
	ui := NewUIManager(websocketURL, username)

	// Set up system tray
	if desk, ok := ui.App.(desktop.App); ok {
		desk.SetSystemTrayIcon(fyne.NewStaticResource("appIcon", icon.AppIcon.Content()))

		// Create menu items
		showItem := fyne.NewMenuItem("Show", func() {
			ui.MainWindow.Show()
		})

		aboutItem := fyne.NewMenuItem("About", func() {
			// Always create a new About window to ensure content is properly initialized
			ui.AboutWindow = NewAboutWindow(ui)
			ui.AboutWindow.Show()
		})

		quitItem := fyne.NewMenuItem("Quit", func() {
			ui.App.Quit()
		})

		connectItem := fyne.NewMenuItem("Connect", func() {
			if ui.VPN != nil {
				// The Run method handles the connection logic
				go ui.VPN.Run(websocketURL, ui.RealtimeData, ui.refreshNetworkList, ui.refreshUI)
			}
		})

		disconnectItem := fyne.NewMenuItem("Disconnect", func() {
			if ui.VPN != nil && ui.VPN.NetworkManager != nil {
				go ui.VPN.NetworkManager.Disconnect()
			}
		})

		// Create the menu with separators for better organization
		menu := fyne.NewMenu("GoVPN",
			showItem,
			fyne.NewMenuItemSeparator(),
			connectItem,
			disconnectItem,
			fyne.NewMenuItemSeparator(),
			aboutItem,
			quitItem,
		)
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

	ui.Run(websocketURL)
	tidyUp()
}

// setupDataPath creates necessary directories for the application
func setupDataPath() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}

	// Create data directory if it doesn't exist
	dataPath := filepath.Join(homeDir, ".govpn")
	err = os.MkdirAll(dataPath, 0755)
	if err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
}

func tidyUp() {
	fmt.Println("Exited")
}
