package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/dialogs"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// UIManager represents the UI manager for the VPN client app
type UIManager struct {
	App                 fyne.App
	MainWindow          fyne.Window
	VPN                 *VPNClient
	ConfigManager       *ConfigManager
	NetworkListComp     *NetworkListComponent
	HomeTabComponent    *HomeTabComponent
	HeaderComponent     *HeaderComponent
	AboutWindow         *AboutWindow
	ConnectDialog       *dialogs.ConnectDialog
	ComputerList        []Computer
	SelectedNetwork     *storage.Network
	defaultWebsocketURL string

	// Nova camada de dados em tempo real
	RealtimeData *data.RealtimeDataLayer
}

// NewUIManager creates a new instance of UIManager
func NewUIManager(websocketURL string, computername string) *UIManager {
	ui := &UIManager{
		defaultWebsocketURL: websocketURL,
	}

	// Criar a camada de dados em tempo real - ensure this is properly initialized
	ui.RealtimeData = data.NewRealtimeDataLayer()

	// Create new app
	ui.App = app.NewWithID("com.itxtoledo.govpn")

	// Initialize configuration manager
	ui.ConfigManager = NewConfigManager()

	// Create main window
	ui.MainWindow = ui.App.NewWindow("GoVPN")
	ui.MainWindow.Resize(fyne.NewSize(300, 600))
	ui.MainWindow.SetMaster()

	// Create VPN client - note the order change to avoid circular reference
	ui.VPN = NewVPNClient(ui.ConfigManager, websocketURL, computername)

	// Initialize default values AFTER all components are created
	ui.RealtimeData.InitDefaults()

	// Setup components
	ui.setupComponents()

	// Setup NetworkManager for VPN client now that dependencies are available
	ui.VPN.SetupNetworkManager(ui.RealtimeData, ui.refreshNetworkList, ui.refreshUI)

	// Configure quit handler
	ui.MainWindow.SetOnClosed(func() {
		ui.handleAppQuit()
	})

	// Configurar listener de eventos da camada de dados em tempo real
	go ui.listenForDataEvents()

	// Refresh UI
	ui.refreshUI()

	return ui
}

// createWindow cria uma nova janela usando o app da UI principal
func (ui *UIManager) createWindow(title string, width, height float32) fyne.Window {
	window := ui.App.NewWindow(title)
	window.Resize(fyne.NewSize(width, height))
	window.SetFixedSize(true)

	// Ensure this window isn't modal which could block the main UI
	window.CenterOnScreen()

	return window
}

// listenForDataEvents escuta eventos da camada de dados em tempo real
func (ui *UIManager) listenForDataEvents() {
	for event := range ui.RealtimeData.Subscribe() {
		switch event.Type {
		case data.EventConnectionStateChanged:
			// Atualizar a UI quando o estado da conexão mudar
			ui.refreshUI()
		case data.EventNetworkJoined:
			// Atualizar a UI quando entrar em uma sala
			ui.refreshNetworkList()
		case data.EventNetworkLeft:
			// Atualizar a UI quando sair de uma sala
			ui.refreshNetworkList()
		case data.EventNetworkDeleted:
			// Atualizar a UI quando uma sala for excluída
			ui.refreshNetworkList()
		case data.EventSettingsChanged:
			// Atualizar configurações quando forem alteradas
			ui.refreshUI()
		case data.EventError:
			// Exibir erro
			log.Printf("Error event: %s", event.Message)
		}
	}
}

// setupComponents initializes all UI components
func (ui *UIManager) setupComponents() {
	// Create components
	ui.HeaderComponent = NewHeaderComponent(ui, ui.defaultWebsocketURL)
	ui.NetworkListComp = NewNetworkListComponent(ui)
	ui.HomeTabComponent = NewHomeTabComponent(ui.ConfigManager, ui.RealtimeData, ui.App, ui.refreshUI, ui.NetworkListComp, ui)

	// Create main container
	headerContainer := ui.HeaderComponent.CreateHeaderContainer()
	tabContainer := ui.HomeTabComponent.CreateHomeTabContainer()

	// Create vertical container
	mainContainer := container.NewBorder(
		headerContainer,
		nil,
		nil,
		nil,
		tabContainer,
	)

	// Set content
	ui.MainWindow.SetContent(container.NewPadded(mainContainer))
}

// handleAppQuit handles application quit
func (ui *UIManager) handleAppQuit() {
	log.Println("Quitting app...")
}

// refreshUI refreshes the UI components
func (ui *UIManager) refreshUI() {
	// Use dados da camada de dados em tempo real para atualizar a UI
	isConnected, _ := ui.RealtimeData.IsConnected.Get()

	// Atualizar componentes baseados nos dados em tempo real
	ui.VPN.IsConnected = isConnected

	// Update header components
	ui.HeaderComponent.updatePowerButtonState()
	ui.HeaderComponent.updateComputerName()
	ui.HeaderComponent.updateNetworkName()
	ui.HeaderComponent.updateBackendStatus()

	// Force refresh widgets
	if ui.MainWindow.Content() != nil {
		ui.MainWindow.Content().Refresh()
	}
}

// GetSelectedNetwork implementa a interface ConnectDialogManager
func (ui *UIManager) GetSelectedNetwork() *storage.Network {
	return ui.SelectedNetwork
}

// ConnectToNetwork implementa a interface ConnectDialogManager
func (ui *UIManager) ConnectToNetwork(networkID, computername string) error {
	// Ensure NetworkManager is initialized
	if ui.VPN == nil || ui.VPN.NetworkManager == nil {
		return fmt.Errorf("network manager not initialized")
	}

	currentNetworkID := ui.VPN.NetworkManager.NetworkID

	// If already connected to the selected network, disconnect
	if currentNetworkID == networkID {
		log.Printf("Attempting to disconnect from network %s", networkID)
		return ui.VPN.NetworkManager.DisconnectNetwork(networkID)
	}

	// If connected to a different network, disconnect first
	if currentNetworkID != "" {
		log.Printf("Already connected to network %s, disconnecting before connecting to %s", currentNetworkID, networkID)
		err := ui.VPN.NetworkManager.DisconnectNetwork(currentNetworkID)
		if err != nil {
			return fmt.Errorf("failed to disconnect from current network: %v", err)
		}
	}

	// Connect to the selected network
	log.Printf("Attempting to connect to network %s", networkID)
	return ui.VPN.NetworkManager.ConnectNetwork(networkID)
}

// refreshNetworkList refreshes the network tree
func (ui *UIManager) refreshNetworkList() {
	// No need to load from database anymore, UI.Networks is maintained in memory

	// Update network tree component
	if ui.NetworkListComp != nil {
		ui.NetworkListComp.UpdateNetworkList()
	}

	// Update UI
	ui.refreshUI()
}

// Run runs the application
func (ui *UIManager) Run(defaultWebsocketURL string) {
	log.Println("Iniciando GoVPN Client")

	// Networks are now managed by RealtimeDataLayer

	// Garantir que as configurações sejam aplicadas antes de exibir a janela
	if ui.VPN != nil {
		// Carrega as configurações do ConfigManager para a camada de dados
		ui.VPN.loadSettings(ui.RealtimeData, ui.App)
	}

	// Verificar o tamanho da janela principal - fixar em 300x600 conforme requisitos
	ui.MainWindow.Resize(fyne.NewSize(300, 600))
	ui.MainWindow.SetFixedSize(true)

	// Configurar tema baseado nas configurações
	config := ui.ConfigManager.GetConfig()
	log.Printf("Applying theme: %s", config.Theme)

	// Atualizar a interface antes de exibir
	ui.refreshUI()

	// Iniciar conexão com o servidor de sinalização em segundo plano
	if ui.VPN != nil {
		go func() {
			fyne.Do(func() {

				log.Println("Iniciando conexão automática com o servidor de sinalização")
				ui.VPN.Run(defaultWebsocketURL, ui.RealtimeData, ui.refreshNetworkList, ui.refreshUI)
			})
		}()
	}

	// Exibir a janela e executar o loop de eventos principal
	ui.MainWindow.ShowAndRun()
}
