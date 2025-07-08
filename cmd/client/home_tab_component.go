// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/home_tab_component.go
package main

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/storage"
	"github.com/itxtoledo/govpn/libs/models"
)

// NetworkManagerAdapter adapts NetworkManager to implement dialogs.NetworkManagerInterface
type NetworkManagerAdapter struct {
	*NetworkManager
}

// CreateRoom adapts the NetworkManager CreateRoom method to match the interface
func (nma *NetworkManagerAdapter) CreateRoom(name, password string) (*models.CreateRoomResponse, error) {
	// Call the original CreateRoom method
	err := nma.NetworkManager.CreateRoom(name, password)
	if err != nil {
		return nil, err
	}

	// Return a response with the current room info
	return &models.CreateRoomResponse{
		RoomID:   nma.NetworkManager.RoomID,
		RoomName: name,
		Password: password,
	}, nil
}

// JoinRoom adapts the NetworkManager JoinRoom method to match the interface
func (nma *NetworkManagerAdapter) JoinRoom(roomID, password, username string) (*models.JoinRoomResponse, error) {
	// Call the original JoinRoom method
	err := nma.NetworkManager.JoinRoom(roomID, password, username)
	if err != nil {
		return nil, err
	}

	// Return a response with the current room info
	return &models.JoinRoomResponse{
		RoomID:   roomID,
		RoomName: roomID, // We don't have room name in the NetworkManager
	}, nil
}

// GetRoomID returns the current room ID
func (nma *NetworkManagerAdapter) GetRoomID() string {
	return nma.NetworkManager.RoomID
}

// GetRealtimeData returns the RealtimeDataLayer
func (nma *NetworkManagerAdapter) GetRealtimeData() *data.RealtimeDataLayer {
	return nma.NetworkManager.RealtimeData
}

// HomeTabComponent representa o componente da aba principal
type HomeTabComponent struct {
	SettingsTab       *SettingsTabComponent
	AppTabs           *container.AppTabs
	NetworksContainer *fyne.Container

	// Dependencies
	ConfigManager   *ConfigManager
	RealtimeData    *data.RealtimeDataLayer
	App             fyne.App
	refreshUI       func()
	NetworkListComp *NetworkListComponent // Add NetworkListComp here
	UI              *UIManager
}

// NewHomeTabComponent cria uma nova instância do componente da aba principal
func NewHomeTabComponent(configManager *ConfigManager, realtimeData *data.RealtimeDataLayer, app fyne.App, refreshUI func(), networkListComp *NetworkListComponent, ui *UIManager) *HomeTabComponent {
	htc := &HomeTabComponent{
		ConfigManager:   configManager,
		RealtimeData:    realtimeData,
		App:             app,
		refreshUI:       refreshUI,
		NetworkListComp: networkListComp,
		UI:              ui,
	}

	// Inicializar a aba de configurações
	htc.SettingsTab = NewSettingsTabComponent(configManager, realtimeData, app, refreshUI)

	return htc
}

// CreateHomeTabContainer cria o container principal da aba inicial
func (htc *HomeTabComponent) CreateHomeTabContainer() *fyne.Container {
	// Criar o container de salas disponíveis
	roomsContainer := htc.NetworkListComp.GetContainer()
	htc.NetworksContainer = roomsContainer

	// Criar um botão para criar uma nova sala
	createRoomButton := widget.NewButtonWithIcon("Create Room", theme.ContentAddIcon(), func() {
		log.Println("Create room button clicked")

		// Check network connection status
		isConnected, _ := htc.RealtimeData.IsConnected.Get()
		if !isConnected {
			dialog.ShowError(fmt.Errorf("not connected to server"), htc.UI.MainWindow)
			return
		}

		// Get username, handling the multiple return values
		username, err := htc.UI.RealtimeData.Username.Get()
		if err != nil {
			log.Printf("Error getting username: %v", err)
			username = "User" // Default fallback
		}

		// Create and show the room creation window (singleton pattern)
		if globalRoomWindow != nil && globalRoomWindow.isOpen && globalRoomWindow.Window != nil {
			// Focus on existing window if already open
			globalRoomWindow.Window.RequestFocus()
			return
		}

		adapter := &NetworkManagerAdapter{htc.UI.VPN.NetworkManager}
		globalRoomWindow = NewRoomWindow(
			htc.UI.App,
			htc.UI.MainWindow,
			adapter.CreateRoom,
			func(roomID, name, password string) error {
				// This is where you'd save the room to local storage if needed
				// For now, we'll just add it to the RealtimeDataLayer
				room := &storage.Room{
					ID:            roomID,
					Name:          name,
					LastConnected: time.Now(),
				}
				adapter.GetRealtimeData().AddRoom(room)
				return nil
			},
			adapter.GetRoomID,
			username,
		)
		globalRoomWindow.Show(ValidatePassword, ConfigurePasswordEntry)
	})

	// Criar um botão para entrar em uma sala
	joinRoomButton := widget.NewButtonWithIcon("Join Room", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "mail-reply"), func() {
		log.Println("Join room button clicked")

		// Check network connection status
		isConnected, _ := htc.RealtimeData.IsConnected.Get()
		if !isConnected {
			dialog.ShowError(fmt.Errorf("not connected to server"), htc.UI.MainWindow)
			return
		}

		// Get username, handling the multiple return values
		username, err := htc.UI.RealtimeData.Username.Get()
		if err != nil {
			log.Printf("Error getting username: %v", err)
			username = "User" // Default fallback
		}

		// Create and show the room joining window (singleton pattern)
		if globalJoinWindow != nil && globalJoinWindow.isOpen && globalJoinWindow.Window != nil {
			// Focus on existing window if already open
			globalJoinWindow.Window.RequestFocus()
			return
		}

		adapter := &NetworkManagerAdapter{htc.UI.VPN.NetworkManager}
		globalJoinWindow = NewJoinWindow(
			htc.UI.App,
			htc.UI.MainWindow,
			adapter.JoinRoom,
			func(roomID, name, password string) error {
				// Save room logic if needed
				return nil
			},
			username,
		)
		globalJoinWindow.Show(ValidatePassword, ConfigurePasswordEntry)
	})

	// Criar o container da aba de salas
	networksTabContent := container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), joinRoomButton, createRoomButton),
		nil,
		nil,
		roomsContainer,
	)

	// Criar o container da aba de configurações
	settingsTabContent := htc.SettingsTab.CreateSettingsContainer()

	// Criar o AppTabs com múltiplas abas
	htc.AppTabs = container.NewAppTabs(
		container.NewTabItem("Networks", networksTabContent),
		container.NewTabItem("Settings", settingsTabContent),
	)

	// Definir a posição das abas na parte superior (abaixo do cabeçalho)
	htc.AppTabs.SetTabLocation(container.TabLocationTop)

	// Envolva o AppTabs em um container para retornar o tipo correto *fyne.Container
	return container.NewStack(htc.AppTabs)
}
