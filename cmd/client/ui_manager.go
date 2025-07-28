package main

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"

	"github.com/itxtoledo/govpn/cmd/client/data"
	dialogs "github.com/itxtoledo/govpn/cmd/client/dialogs"
	smodels "github.com/itxtoledo/govpn/libs/signaling/models"
)

// UIManager represents the UI manager for the VPN client app
type UIManager struct {
	App                 fyne.App
	MainWindow          fyne.Window
	VPN                 *VPNClient
	ConfigManager       *ConfigManager
	NetworkListComp     *NetworkListComponent
	HomeScreenComponent *HomeScreenComponent
	HeaderComponent     *HeaderComponent
	AboutWindow         *AboutWindow
	ConnectDialog       *dialogs.ConnectDialog
	ComputerList        []smodels.Computer
	SelectedNetwork     *data.Network
	defaultWebsocketURL string

	// Nova camada de dados em tempo real
	RealtimeData *data.RealtimeDataLayer
}

// NewUIManager creates a new instance of UIManager
func NewUIManager(websocketURL string, computername string, configPath string) *UIManager {
	ui := &UIManager{
		defaultWebsocketURL: websocketURL,
	}

	// Criar a camada de dados em tempo real - ensure this is properly initialized
	ui.RealtimeData = data.NewRealtimeDataLayer()

	// Create new app
	ui.App = app.NewWithID("com.itxtoledo.govpn")

	// Initialize configuration manager
	ui.ConfigManager = NewConfigManager(configPath)

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
		case data.EventNetworksChanged:
			// Atualizar a UI quando a lista de redes for alterada
			ui.refreshNetworkList()
		case data.EventSettingsChanged:
			// Atualizar configurações quando forem alteradas
			ui.refreshUI()
		case data.EventComputerJoined:
			// Exibir notificação de computador conectado
			dialog.ShowInformation("Computer Connected", event.Message, ui.MainWindow)
		case data.EventComputerConnected:
			// Exibir notificação de computador conectado
			dialog.ShowInformation("Computer Connected", event.Message, ui.MainWindow)
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
	ui.HomeScreenComponent = NewHomeScreenComponent(ui.ConfigManager, ui.RealtimeData, ui.NetworkListComp, ui)

	// Create main container
	headerContainer := ui.HeaderComponent.CreateHeaderContainer()

	// Create vertical container
	mainContainer := container.NewBorder(
		headerContainer,
		nil,
		nil,
		nil,
		ui.HomeScreenComponent.CreateHomeScreenContainer(),
	)

	// Set content
	ui.MainWindow.SetContent(container.NewPadded(mainContainer))
}

// ShowSettingsWindow creates and shows the settings window
// OpenChatWindow creates and shows the chat window
func (ui *UIManager) OpenChatWindow(network *data.Network) {
	// Create and show the chat window (singleton pattern per network)
	// For simplicity, let's assume only one chat window can be open at a time for now.
	// In a real application, you might want a map of chat windows per network ID.
	if globalChatWindow != nil && globalChatWindow.BaseWindow.Window != nil {
		// Focus on existing window if already open
		globalChatWindow.BaseWindow.Window.RequestFocus()
		return
	}

	globalChatWindow = NewChatWindow(
		ui.App,
		network,
		ui.VPN.WebRTCManager, // Pass the WebRTCManager
	)
	globalChatWindow.Show()
}

func (ui *UIManager) ShowAboutWindow() {
	// Create and show the about window (singleton pattern)
	if ui.AboutWindow != nil && ui.AboutWindow.BaseWindow.Window != nil {
		// Focus on existing window if already open
		ui.AboutWindow.BaseWindow.Window.RequestFocus()
		return
	}

	publicKey, _ := ui.ConfigManager.GetKeyPair()

	ui.AboutWindow = NewAboutWindow(
		ui.App,
		publicKey,
	)
	ui.AboutWindow.Show()
}

func (ui *UIManager) ShowSettingsWindow() {
	config := ui.ConfigManager.GetConfig()

	// Create and show the settings window (singleton pattern)
	if globalSettingsWindow != nil && globalSettingsWindow.BaseWindow.Window != nil {
		// Focus on existing window if already open
		globalSettingsWindow.BaseWindow.Window.RequestFocus()
		return
	}

	globalSettingsWindow = NewSettingsWindow(
		ui.App,
		ui.ConfigManager,
		config,
		ui.HandleSettingsSaved,
	)
	globalSettingsWindow.Show()
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

	// Force refresh widgets
	fyne.Do(func() {
		if ui.MainWindow.Content() != nil {
			ui.MainWindow.Content().Refresh()
		}
	})
}

// GetSelectedNetwork implementa a interface ConnectDialogManager
func (ui *UIManager) GetSelectedNetwork() *data.Network {
	return ui.SelectedNetwork
}

// GetMainWindow implements the ConnectDialogManager interface
func (ui *UIManager) GetMainWindow() fyne.Window {
	return ui.MainWindow
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
	fyne.Do(func() {
		if ui.NetworkListComp != nil {
			ui.NetworkListComp.UpdateNetworkList()
		}
	})

	// Update UI
	ui.refreshUI()
}

// HandleNetworkCreated is the callback for when a network is created
func (ui *UIManager) HandleNetworkCreated(networkID, networkName, pin string) {
	network := data.Network{
		NetworkID:     networkID,
		NetworkName:   networkName,
		LastConnected: time.Now(),
	}
	ui.RealtimeData.AddNetwork(network)
	ui.RealtimeData.EmitEvent(data.EventNetworkCreated, "Network created", network)

	dialog.ShowInformation("Success", "Network created and saved!", ui.MainWindow)
}

// HandleNetworkJoined is the callback for when a network is joined
func (ui *UIManager) HandleNetworkJoined(networkID, pin string) {
	network := data.Network{
		NetworkID:     networkID,
		NetworkName:   networkID, // Name is the ID for joined networks
		LastConnected: time.Now(),
	}
	ui.RealtimeData.AddNetwork(network)

	dialog.ShowInformation("Success!", "Successfully joined the network!", ui.MainWindow)
}

// Run runs the application
func (ui *UIManager) HandleSettingsSaved(config Config) {
	// Save new settings
	err := ui.ConfigManager.UpdateConfig(config)
	if err != nil {
		log.Printf("Error saving settings: %v", err)
	}

	// Apply settings
	ui.applySettings(config)
}

// applySettings applies the settings
func (ui *UIManager) applySettings(config Config) {

	// Update computer name in realtime data layer
	ui.RealtimeData.SetComputerName(config.ComputerName)

	// Update server address
	ui.RealtimeData.SetServerAddress(config.ServerAddress)

	// Send updated client info to the server
	if ui.VPN != nil && ui.VPN.NetworkManager != nil {
		// Update client info on the server only if the computer name has changed
		if config.ComputerName != ui.ConfigManager.GetConfig().ComputerName {
			ui.VPN.NetworkManager.UpdateClientInfo()
		}
	}

	// Emit settings changed event
	ui.RealtimeData.EmitEvent(data.EventSettingsChanged, "Settings updated", nil)

	// Refresh UI
}

// Run starts the UI and the VPN client
func (ui *UIManager) Run(defaultWebsocketURL string) {
	log.Println("Iniciando GoVPN Client")

	// Networks are now managed by RealtimeDataLayer

	// Garantir que as configurações sejam aplicadas antes de exibir a janela
	if ui.VPN != nil {
		// Carrega as configurações do ConfigManager para a camada de dados
		ui.VPN.loadSettings(ui.RealtimeData)
	}

	// Verificar o tamanho da janela principal - fixar em 300x600 conforme requisitos
	ui.MainWindow.Resize(fyne.NewSize(300, 600))
	ui.MainWindow.SetFixedSize(true)

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
