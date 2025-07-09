// filepath: /Computers/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/home_tab_component.go
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

// CreateNetwork adapts the NetworkManager CreateNetwork method to match the interface
func (nma *NetworkManagerAdapter) CreateNetwork(name, password string) (*models.CreateNetworkResponse, error) {
	// Call the original CreateNetwork method
	err := nma.NetworkManager.CreateNetwork(name, password)
	if err != nil {
		return nil, err
	}

	// Return a response with the current network info
	return &models.CreateNetworkResponse{
		NetworkID:   nma.NetworkManager.NetworkID,
		NetworkName: name,
		Password:    password,
	}, nil
}

// JoinNetwork adapts the NetworkManager JoinNetwork method to match the interface
func (nma *NetworkManagerAdapter) JoinNetwork(networkID, password, computername string) (*models.JoinNetworkResponse, error) {
	// Call the original JoinNetwork method
	err := nma.NetworkManager.JoinNetwork(networkID, password, computername)
	if err != nil {
		return nil, err
	}

	// Return a response with the current network info
	return &models.JoinNetworkResponse{
		NetworkID:   networkID,
		NetworkName: networkID, // We don't have network name in the NetworkManager
	}, nil
}

// GetNetworkID returns the current network ID
func (nma *NetworkManagerAdapter) GetNetworkID() string {
	return nma.NetworkManager.NetworkID
}

// GetRealtimeData returns the RealtimeDataLayer
func (nma *NetworkManagerAdapter) GetRealtimeData() *data.RealtimeDataLayer {
	return nma.NetworkManager.RealtimeData
}

// HomeTabComponent representa o componente da aba principal
type HomeTabComponent struct {
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

	return htc
}

// CreateHomeTabContainer cria o container principal da aba inicial
func (htc *HomeTabComponent) CreateHomeTabContainer() *fyne.Container {
	// Criar o container de salas disponíveis
	networksContainer := htc.NetworkListComp.GetContainer()
	htc.NetworksContainer = networksContainer

	// Criar um botão para criar uma nova sala
	createNetworkButton := widget.NewButtonWithIcon("Create Network", theme.ContentAddIcon(), func() {
		log.Println("Create network button clicked")

		// Check network connection status
		isConnected, _ := htc.RealtimeData.IsConnected.Get()
		if !isConnected {
			dialog.ShowError(fmt.Errorf("not connected to server"), htc.UI.MainWindow)
			return
		}

		// Get computername, handling the multiple return values
		computername, err := htc.UI.RealtimeData.ComputerName.Get()
		if err != nil {
			log.Printf("Error getting computername: %v", err)
			computername = "Computer" // Default fallback
		}

		// Create and show the network creation window (singleton pattern)
		if globalNetworkWindow != nil && globalNetworkWindow.BaseWindow.Window != nil {
			// Focus on existing window if already open
			globalNetworkWindow.BaseWindow.Window.RequestFocus()
			return
		}

		adapter := &NetworkManagerAdapter{htc.UI.VPN.NetworkManager}
		globalNetworkWindow = NewNetworkWindow(
			htc.UI,
			adapter.CreateNetwork,
			func(networkID, name, password string) error {
				// This is where you'd save the network to local storage if needed
				// For now, we'll just add it to the RealtimeDataLayer
				network := &storage.Network{
					ID:            networkID,
					Name:          name,
					LastConnected: time.Now(),
				}
				adapter.GetRealtimeData().AddNetwork(network)
				return nil
			},
			adapter.GetNetworkID,
			computername,
			ValidatePassword,
			ConfigurePasswordEntry,
		)
		globalNetworkWindow.Show()
	})

	// Criar um botão para entrar em uma sala
	joinNetworkButton := widget.NewButtonWithIcon("Join Network", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "mail-reply"), func() {
		log.Println("Join network button clicked")

		// Check network connection status
		isConnected, _ := htc.RealtimeData.IsConnected.Get()
		if !isConnected {
			dialog.ShowError(fmt.Errorf("not connected to server"), htc.UI.MainWindow)
			return
		}

		// Get computername, handling the multiple return values
		computername, err := htc.UI.RealtimeData.ComputerName.Get()
		if err != nil {
			log.Printf("Error getting computername: %v", err)
			computername = "Computer" // Default fallback
		}

		// Create and show the network joining window (singleton pattern)
		if globalJoinWindow != nil && globalJoinWindow.BaseWindow.Window != nil {
			// Focus on existing window if already open
			globalJoinWindow.BaseWindow.Window.RequestFocus()
			return
		}

		adapter := &NetworkManagerAdapter{htc.UI.VPN.NetworkManager}
		globalJoinWindow = NewJoinWindow(
			htc.UI,
			adapter.JoinNetwork,
			func(networkID, name, password string) error {
				// Save network logic if needed
				return nil
			},
			computername,
			ValidatePassword,
			ConfigurePasswordEntry,
		)
		globalJoinWindow.Show()
	})

	// Criar o container da aba de salas
	networksTabContent := container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), joinNetworkButton, createNetworkButton),
		nil,
		nil,
		networksContainer,
	)

	// Criar o AppTabs com múltiplas abas
	htc.AppTabs = container.NewAppTabs(
		container.NewTabItem("Networks", networksTabContent),
	)

	// Definir a posição das abas na parte superior (abaixo do cabeçalho)
	htc.AppTabs.SetTabLocation(container.TabLocationTop)

	// Envolva o AppTabs em um container para retornar o tipo correto *fyne.Container
	return container.NewStack(htc.AppTabs)
}
