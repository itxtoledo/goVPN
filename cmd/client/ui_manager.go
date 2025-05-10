package main

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// UIManager represents the UI manager for the VPN client app
type UIManager struct {
	App               fyne.App
	MainWindow        fyne.Window
	VPN               *VPNClient
	ConfigManager     *ConfigManager
	NetworkListComp   *NetworkListComponent
	RoomItemComponent *RoomItemComponent
	HomeTabComponent  *HomeTabComponent
	HeaderComponent   *HeaderComponent
	AboutWindow       *AboutWindow
	ConnectDialog     *ConnectDialog
	Rooms             []*storage.Room
	ComputerList      []Computer
	SelectedRoom      *storage.Room

	// Nova camada de dados em tempo real
	RealtimeData *data.RealtimeDataLayer
}

// NewUIManager creates a new instance of UIManager
func NewUIManager() *UIManager {
	ui := &UIManager{}

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
	ui.VPN = NewVPNClient(ui)

	// Initialize default values AFTER all components are created
	ui.RealtimeData.InitDefaults()

	// Setup components
	ui.setupComponents()

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
		case data.EventRoomJoined:
			// Atualizar a UI quando entrar em uma sala
			ui.refreshNetworkList()
		case data.EventRoomLeft:
			// Atualizar a UI quando sair de uma sala
			ui.refreshNetworkList()
		case data.EventRoomDeleted:
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
	ui.HeaderComponent = NewHeaderComponent(ui)
	ui.NetworkListComp = NewNetworkListComponent(ui)
	ui.RoomItemComponent = NewRoomItemComponent(ui)
	ui.HomeTabComponent = NewHomeTabComponent(ui)

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
	ui.HeaderComponent.updateIPInfo()
	ui.HeaderComponent.updateUsername()
	ui.HeaderComponent.updateRoomName()
	ui.HeaderComponent.updateBackendStatus()

	// Force refresh widgets
	if ui.MainWindow.Content() != nil {
		ui.MainWindow.Content().Refresh()
	}
}

// refreshNetworkList refreshes the network tree
func (ui *UIManager) refreshNetworkList() {
	// No need to load from database anymore, UI.Rooms is maintained in memory

	// Update network tree component
	if ui.NetworkListComp != nil {
		ui.NetworkListComp.updateNetworkList()
	}

	// Update UI
	ui.refreshUI()
}

// Run runs the application
func (ui *UIManager) Run() {
	log.Println("Iniciando GoVPN Client")

	// Initialize the room list as an empty slice if it's nil
	if ui.Rooms == nil {
		ui.Rooms = make([]*storage.Room, 0)
	}

	// Garantir que as configurações sejam aplicadas antes de exibir a janela
	if ui.VPN != nil {
		// Carrega as configurações do ConfigManager para a camada de dados
		ui.VPN.loadSettings()
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
				// Pequeno atraso para garantir que a UI esteja pronta
				time.Sleep(500 * time.Millisecond)
				log.Println("Iniciando conexão automática com o servidor de sinalização")
				ui.VPN.Run()
			})
		}()
	}

	// Exibir a janela e executar o loop de eventos principal
	ui.MainWindow.ShowAndRun()
}
