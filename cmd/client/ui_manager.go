package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// UIManager gerencia a interface gráfica do cliente VPN
type UIManager struct {
	VPN            *VPNClient
	App            fyne.App
	MainWindow     fyne.Window
	RoomWindow     *RoomWindow
	LoginWindow    *LoginWindow
	AboutWindow    *AboutWindow
	SettingsWindow *SettingsWindow
	ConnectWindow  *ConnectWindow

	// Componentes modulares da interface
	HeaderComponent      *HeaderComponent
	NetworkTreeComponent *NetworkTreeComponent
	HomeTabComponent     *HomeTabComponent
	NetworkTabComponent  *NetworkTabComponent
	AboutTabComponent    *AboutTabComponent
	SettingsTabComponent *SettingsTabComponent

	// Referências diretas para componentes importantes mantidas para compatibilidade
	PowerButton   *widget.Button
	NetworkList   *widget.Tree
	IPInfoLabel   *widget.Label
	RoomNameLabel *widget.Label
	NetworkUsers  map[string]bool // Map das chaves públicas (truncadas) e seu status (online/offline)

	// Interface elements for tabbed interface
	TabContainer *container.AppTabs
}

// NewUIManager cria um novo gerenciador de interface gráfica
func NewUIManager(vpn *VPNClient) *UIManager {
	uiManager := &UIManager{
		VPN:          vpn,
		App:          app.New(),
		NetworkUsers: make(map[string]bool),
	}

	// Configura o tema
	uiManager.App.Settings().SetTheme(theme.DarkTheme())

	// Cria a janela principal
	uiManager.MainWindow = uiManager.App.NewWindow("goVPN")

	// Configura tamanho fixo e define como janela principal
	uiManager.MainWindow.SetFixedSize(true)
	uiManager.MainWindow.SetMaster()

	// Define o tamanho exato da janela
	uiManager.MainWindow.Resize(fyne.NewSize(300, 600))

	// Configura o manipulador de fechamento da janela
	uiManager.MainWindow.SetCloseIntercept(func() {
		if vpn.IsConnected {
			// Se estiver conectado, desconecta antes de fechar
			vpn.NetworkManager.LeaveRoom()
		}
		uiManager.MainWindow.Close()
	})

	return uiManager
}

// Start inicia a interface gráfica
func (ui *UIManager) Start() {
	// Inicialização prévia de componentes essenciais
	ui.preInitializeComponents()

	// Configura o menu principal
	ui.setupMainMenu()

	// Configura a interface principal
	ui.setupMainInterface()

	// Mostra a janela principal
	ui.MainWindow.ShowAndRun()
}

// preInitializeComponents inicializa previamente componentes essenciais da interface
func (ui *UIManager) preInitializeComponents() {
	// Garante que o NetworkManager está pronto antes de prosseguir
	if ui.VPN == nil || ui.VPN.NetworkManager == nil {
		log.Println("Aviso: VPN ou NetworkManager não inicializados corretamente")
		return
	}

	// Inicializa apenas os componentes necessários para a tela inicial
	// Outros componentes serão inicializados sob demanda
	if ui.RoomWindow == nil {
		ui.RoomWindow = NewRoomWindow(ui)
	}
}

// setupMainMenu configura o menu da aplicação
func (ui *UIManager) setupMainMenu() {
	systemMenu := fyne.NewMenu("System",
		fyne.NewMenuItem("Conectar", func() {
			// Certifique-se de que a janela de conexão está inicializada
			if ui.ConnectWindow == nil {
				ui.ConnectWindow = NewConnectWindow(ui)
			}
			ui.ConnectWindow.Show()
		}),
		fyne.NewMenuItem("Desconectar", func() {
			if ui.VPN.IsConnected {
				err := ui.VPN.NetworkManager.LeaveRoom()
				if err != nil {
					log.Printf("Erro ao desconectar: %v", err)
				}
				ui.VPN.IsConnected = false
				ui.updatePowerButtonState()
				ui.updateIPInfo()
				ui.updateRoomName()
				ui.refreshNetworkList()
			}
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Sair", func() {
			ui.MainWindow.Close()
		}),
	)

	mainMenu := fyne.NewMainMenu(
		systemMenu,
	)

	ui.MainWindow.SetMainMenu(mainMenu)
}

// setupMainInterface configura a interface principal
func (ui *UIManager) setupMainInterface() {
	// Inicializa os componentes da interface de usuário
	ui.initializeUIComponents()

	// Cria o container de tabs usando os componentes
	ui.setupTabs()

	// Obtém o cabeçalho do componente Header
	header := ui.HeaderComponent.CreateHeaderContainer()

	// Limita o tamanho máximo dos containers para evitar expansão da janela
	maxWidth := float32(300)

	// Define tamanho máximo para o tab container
	tabContainer := container.NewMax(ui.TabContainer)
	tabContainer.Resize(fyne.NewSize(maxWidth, 540)) // Altura aproximada para tabs

	// Container principal com dimensões fixas
	mainContent := container.New(
		layout.NewMaxLayout(),
		container.NewBorder(
			header,       // top
			nil,          // bottom
			nil,          // left
			nil,          // right
			tabContainer, // center
		),
	)

	ui.MainWindow.SetContent(mainContent)

	// Atualiza o estado inicial
	ui.updatePowerButtonState()
	ui.updateIPInfo()
	ui.updateRoomName()
}

// initializeUIComponents inicializa todos os componentes da UI
func (ui *UIManager) initializeUIComponents() {
	// Inicializa os componentes na ordem correta de dependência
	ui.HeaderComponent = NewHeaderComponent(ui)
	ui.NetworkTreeComponent = NewNetworkTreeComponent(ui)
	ui.HomeTabComponent = NewHomeTabComponent(ui, ui.NetworkTreeComponent)
	ui.NetworkTabComponent = NewNetworkTabComponent(ui)
	ui.AboutTabComponent = NewAboutTabComponent(ui)
	ui.SettingsTabComponent = NewSettingsTabComponent(ui)

	// Mantém referências diretas para compatibilidade com código existente
	ui.PowerButton = ui.HeaderComponent.PowerButton
	ui.NetworkList = ui.NetworkTreeComponent.NetworkList
	ui.IPInfoLabel = ui.HeaderComponent.IPInfoLabel
	ui.RoomNameLabel = ui.HeaderComponent.RoomNameLabel
}

// setupTabs configura as tabs usando os componentes
func (ui *UIManager) setupTabs() {
	// Cria o container de tabs com tamanho fixo
	ui.TabContainer = container.NewAppTabs(
		container.NewTabItem("Home", ui.HomeTabComponent.GetContainer()),
		container.NewTabItem("Network", ui.NetworkTabComponent.GetContainer()),
		// About and Settings tabs removed as they're now accessible from header buttons
	)

	// Limita o tamanho máximo do contêiner de tabs
	ui.TabContainer.SetTabLocation(container.TabLocationTop)
	maxSize := fyne.NewSize(300, 520)
	container.NewWithoutLayout(ui.TabContainer).Resize(maxSize)
}

// updatePowerButtonState atualiza o estado visual do botão de ligar/desligar
func (ui *UIManager) updatePowerButtonState() {
	ui.HeaderComponent.updatePowerButtonState()
}

// updateIPInfo atualiza as informações de IP exibidas
func (ui *UIManager) updateIPInfo() {
	ui.HeaderComponent.updateIPInfo()
}

// updateRoomName atualiza o nome da sala exibido
func (ui *UIManager) updateRoomName() {
	ui.HeaderComponent.updateRoomName()
}

// refreshNetworkList atualiza a lista de usuários na rede
func (ui *UIManager) refreshNetworkList() {
	// Limpa a lista de usuários atual
	ui.NetworkUsers = make(map[string]bool)

	if ui.VPN.IsConnected && ui.VPN.NetworkManager != nil {
		// Adiciona a própria chave pública do usuário (mostrada de forma truncada)
		ownPublicKey := ui.VPN.getPublicKey()
		if ownPublicKey != "" {
			formattedKey := ui.VPN.NetworkManager.GetFormattedPublicKey(ownPublicKey)
			ui.NetworkUsers[formattedKey] = true
		}

		// Adiciona todos os peers da sala com suas chaves públicas
		for peerPublicKey, isOnline := range ui.VPN.NetworkManager.PeersByPublicKey {
			formattedKey := ui.VPN.NetworkManager.GetFormattedPublicKey(peerPublicKey)
			ui.NetworkUsers[formattedKey] = isOnline
		}
	}

	// Atualiza a visualização da árvore de rede
	if ui.NetworkTreeComponent != nil {
		ui.NetworkTreeComponent.RefreshTree()
	}

	// Atualiza o conteúdo da aba Home
	if ui.HomeTabComponent != nil {
		ui.HomeTabComponent.updateContent()
	}
}

// ShowMessage exibe uma mensagem na interface
func (ui *UIManager) ShowMessage(title, message string) {
	// Usar o diálogo padrão do Fyne
	dialog := widget.NewModalPopUp(
		container.NewVBox(
			widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel(message),
			container.NewHBox(
				widget.NewButton("OK", func() {
					if ui.MainWindow.Canvas().Overlays().Top() != nil {
						ui.MainWindow.Canvas().Overlays().Top().Hide()
					}
				}),
			),
		),
		ui.MainWindow.Canvas(),
	)

	dialog.Show()
}

// createWindow creates and configures a new window with specified parameters
func (ui *UIManager) createWindow(title string, width, height int, resizable bool) fyne.Window {
	window := ui.App.NewWindow(title)
	window.Resize(fyne.NewSize(float32(width), float32(height)))
	window.SetFixedSize(!resizable)

	// Apenas garantimos que a janela seja exibida sobre outras, mas não afete o ciclo de vida da aplicação
	window.SetOnClosed(func() {
		// Janela fechada, mas não encerramos a aplicação
		// A janela será recriada na próxima vez que for necessária
	})

	return window
}
