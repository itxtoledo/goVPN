package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/data"
	st "github.com/itxtoledo/govpn/cmd/client/storage"
	smodels "github.com/itxtoledo/govpn/libs/signaling/models"
)

// NetworkManagerAdapter adapts NetworkManager to implement dialogs.NetworkManagerInterface
type NetworkManagerAdapter struct {
	*NetworkManager
}

// CreateNetwork adapts the NetworkManager CreateNetwork method to match the interface
func (nma *NetworkManagerAdapter) CreateNetwork(name, pin string) (*smodels.CreateNetworkResponse, error) {
	// Call the original CreateNetwork method
	err := nma.NetworkManager.CreateNetwork(name, pin)
	if err != nil {
		return nil, err
	}

	// Return a response with the current network info
	return &smodels.CreateNetworkResponse{
		NetworkID:   nma.NetworkManager.NetworkID,
		NetworkName: name,
		PIN:         pin,
	}, nil
}

// JoinNetwork adapts the NetworkManager JoinNetwork method to match the interface
func (nma *NetworkManagerAdapter) JoinNetwork(networkID, pin, computername string) (*smodels.JoinNetworkResponse, error) {
	// Call the original JoinNetwork method
	err := nma.NetworkManager.JoinNetwork(networkID, pin, computername)
	if err != nil {
		return nil, err
	}

	// Return a response with the current network info
	return &smodels.JoinNetworkResponse{
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

// HomeScreenComponent representa o componente da tela principal
type HomeScreenComponent struct {
	NetworksContainer *fyne.Container

	// Dependencies
	ConfigManager   *st.ConfigManager
	RealtimeData    *data.RealtimeDataLayer
	NetworkListComp *NetworkListComponent // Add NetworkListComp here
	UI              *UIManager
}

// NewHomeScreenComponent cria uma nova instância do componente da tela principal
func NewHomeScreenComponent(configManager *st.ConfigManager, realtimeData *data.RealtimeDataLayer, networkListComp *NetworkListComponent, ui *UIManager) *HomeScreenComponent {
	htc := &HomeScreenComponent{
		ConfigManager:   configManager,
		RealtimeData:    realtimeData,
		NetworkListComp: networkListComp,
		UI:              ui,
	}

	return htc
}

// CreateHomeTabContainer cria o container principal da aba inicial
func (htc *HomeScreenComponent) CreateHomeScreenContainer() *fyne.Container {
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
			htc.UI.App,
			adapter.CreateNetwork,
			adapter.GetNetworkID,
			computername,
			func(networkID, networkName, pin string) {
				htc.UI.HandleNetworkCreated(networkID, networkName, pin)
			},
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
			htc.UI.App,
			adapter.JoinNetwork,
			computername,
			func(networkID, pin string) {
				htc.UI.HandleNetworkJoined(networkID, pin)
			},
		)
		globalJoinWindow.Show()
	})

	// Criar o container da aba de salas
	return container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), joinNetworkButton, createNetworkButton),
		nil,
		nil,
		networksContainer,
	)
}
