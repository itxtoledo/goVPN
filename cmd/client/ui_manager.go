package main

import (
	"image/color"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/services/client/icon"
)

// UIManager manages the VPN client's graphical interface
type UIManager struct {
	VPN            *VPNClient
	App            fyne.App
	MainWindow     fyne.Window
	RoomDialog     *RoomDialog
	ConnectDialog  *ConnectDialog
	LoginWindow    *LoginWindow
	AboutWindow    *AboutWindow
	SettingsWindow *SettingsWindow
	ConfigManager  *ConfigManager

	// Modular interface components
	HeaderComponent      *HeaderComponent
	NetworkTreeComponent *NetworkTreeComponent
	HomeTabComponent     *HomeTabComponent

	// Direct references to important components maintained for compatibility
	PowerButton   *widget.Button
	NetworkList   *widget.Tree
	IPInfoLabel   *canvas.Text // Changed from *widget.Label to *canvas.Text
	RoomNameLabel *widget.Label
	NetworkUsers  map[string]bool // Map of public keys (truncated) and their status (online/offline)

	// Interface elements for tabbed interface
	TabContainer *container.AppTabs
}

// NewUIManager creates a new graphical interface manager
func NewUIManager(vpn *VPNClient) *UIManager {
	uiManager := &UIManager{
		VPN:           vpn,
		App:           app.New(),
		NetworkUsers:  make(map[string]bool),
		ConfigManager: NewConfigManager(),
	}

	// Load saved settings
	config := uiManager.ConfigManager.GetConfig()

	// Configure theme based on saved preferences
	switch config.ThemePreference {
	case "Light":
		uiManager.App.Settings().SetTheme(theme.LightTheme())
	case "Dark":
		uiManager.App.Settings().SetTheme(theme.DarkTheme())
	default:
		uiManager.App.Settings().SetTheme(theme.DefaultTheme())
	}

	// Configure signal server
	if vpn.NetworkManager != nil && config.SignalServer != "" {
		vpn.NetworkManager.SignalServer = config.SignalServer
	}

	// Create main window
	uiManager.MainWindow = uiManager.App.NewWindow("goVPN")

	appIcon := icon.Power

	uiManager.App.SetIcon(appIcon)
	uiManager.MainWindow.SetIcon(appIcon)

	// Set fixed size and define as main window
	uiManager.MainWindow.SetFixedSize(true)
	uiManager.MainWindow.SetMaster()

	// Set exact window size
	uiManager.MainWindow.Resize(fyne.NewSize(300, 600))

	// Configure window close handler
	uiManager.MainWindow.SetCloseIntercept(func() {
		if vpn.IsConnected {
			// If connected, disconnect before closing
			vpn.NetworkManager.LeaveRoom()
		}
		uiManager.MainWindow.Close()
	})

	return uiManager
}

// resolveResourcePath retorna o caminho absoluto para um recurso
func (ui *UIManager) resolveResourcePath(filename string) string {
	// Lista de possíveis localizações para o arquivo
	possiblePaths := []string{
		filename,                       // Caminho relativo atual
		"cmd/client/" + filename,       // A partir da raiz do projeto
		"../client/" + filename,        // Se estiver no diretório cmd
		"../../cmd/client/" + filename, // Se estiver em outro subdiretório
	}

	// Tenta encontrar o arquivo em cada um dos possíveis caminhos
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path // Retorna o primeiro caminho válido
		}
	}

	// Se não encontrar, retorna o caminho original
	return filename
}

// updateSystemTrayStatus atualiza o status do ícone na bandeja do sistema
func (ui *UIManager) updateSystemTrayStatus() {
	// Create the title text based on connection state
	title := "goVPN - Disconnected"
	
	if ui.VPN.IsConnected {
		networkName := "unknown"
		if ui.VPN.NetworkManager != nil && ui.VPN.NetworkManager.RoomName != "" {
			networkName = ui.VPN.NetworkManager.RoomName
		}

		// Mostra o nome da rede e o IP
		ipAddress := "unknown"
		if ui.VPN.NetworkManager != nil && ui.VPN.NetworkManager.VirtualNetwork != nil {
			ipAddress = ui.VPN.NetworkManager.VirtualNetwork.GetLocalIP()
		}

		title = "goVPN - Connected\n" +
			"Network: " + networkName + "\n" +
			"IP: " + ipAddress
	}
	
	// Using Fyne's internal async functionality to ensure UI updates happen on the main thread
	// Store the title locally and update it in the main UI thread
	finalTitle := title
	fyne.CurrentApp().SendNotification(&fyne.Notification{
		Title:   finalTitle,
		Content: "",
	})
	
	// For the window title, we use a callback to the main window
	ui.MainWindow.SetTitle(finalTitle)
}

// refreshNetworkList updates the list of users in the network
func (ui *UIManager) refreshNetworkList() {
	// Clear current user list
	ui.NetworkUsers = make(map[string]bool)

	if ui.VPN.IsConnected && ui.VPN.NetworkManager != nil {
		// Add user's own public key (shown truncated)
		ownPublicKey := ui.VPN.getPublicKey()
		if ownPublicKey != "" {
			formattedKey := ui.VPN.NetworkManager.GetFormattedPublicKey(ownPublicKey)
			ui.NetworkUsers[formattedKey] = true
		}

		// Add all room peers with their public keys
		for peerPublicKey, isOnline := range ui.VPN.NetworkManager.PeersByPublicKey {
			formattedKey := ui.VPN.NetworkManager.GetFormattedPublicKey(peerPublicKey)
			ui.NetworkUsers[formattedKey] = isOnline
		}
	}

	// Update UI elements directly - Fyne handles thread safety internally
	// Update Home tab content
	if ui.HomeTabComponent != nil {
		ui.HomeTabComponent.updateContent()
	}

	// Update username display
	if ui.HeaderComponent != nil {
		ui.HeaderComponent.updateUsername()
	}

	// Atualiza o status na bandeja do sistema
	ui.updateSystemTrayStatus()
}

// updatePowerButtonState updates the visual state of the power button
func (ui *UIManager) updatePowerButtonState() {
	ui.HeaderComponent.updatePowerButtonState()

	// Também atualiza o status na bandeja do sistema
	ui.updateSystemTrayStatus()
}

// setupMainMenu configures the application menu
func (ui *UIManager) setupMainMenu() {
	systemMenu := fyne.NewMenu("System",
		fyne.NewMenuItem("Connect", func() {
			// Make sure the connection window is initialized
			if ui.ConnectDialog == nil {
				ui.ConnectDialog = NewConnectDialog(ui)
			}
			ui.ConnectDialog.Show()
		}),
		fyne.NewMenuItem("Disconnect", func() {
			if ui.VPN.IsConnected {
				err := ui.VPN.NetworkManager.LeaveRoom()
				if err != nil {
					log.Printf("Error disconnecting: %v", err)
				}
				ui.VPN.IsConnected = false
				ui.updatePowerButtonState()
				// ui.updateIPInfo()
				// ui.updateRoomName()
				ui.refreshNetworkList()
			}
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Exit", func() {
			ui.MainWindow.Close()
		}),
	)

	mainMenu := fyne.NewMainMenu(
		systemMenu,
	)

	ui.MainWindow.SetMainMenu(mainMenu)
}

// setupMainInterface configures the main interface
func (ui *UIManager) setupMainInterface() {
	// Initialize user interface components
	ui.initializeUIComponents()

	// Create tabs container using components
	ui.setupTabs()

	// Get header from Header component
	header := ui.HeaderComponent.CreateHeaderContainer()

	// Limit maximum size of containers to prevent window expansion
	maxWidth := float32(300)

	// Set maximum size for tab container
	tabContainer := container.NewMax(ui.TabContainer)
	tabContainer.Resize(fyne.NewSize(maxWidth, 540)) // Approximate height for tabs

	// Main container with fixed dimensions
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

	// Update initial state
	ui.updatePowerButtonState()
	// ui.updateIPInfo()
	// ui.updateRoomName()
}

// initializeUIComponents initializes all UI components
func (ui *UIManager) initializeUIComponents() {
	// Initialize components in correct dependency order
	ui.HeaderComponent = NewHeaderComponent(ui)
	ui.NetworkTreeComponent = NewNetworkTreeComponent(ui)
	ui.HomeTabComponent = NewHomeTabComponent(ui, ui.NetworkTreeComponent)

	// Set username from config
	if ui.VPN.NetworkManager != nil {
		config := ui.ConfigManager.GetConfig()
		ui.VPN.NetworkManager.Username = config.Username
	}

	// Maintain direct references for compatibility with existing code
	ui.PowerButton = ui.HeaderComponent.PowerButton
	ui.NetworkList = ui.NetworkTreeComponent.ui.NetworkList
	ui.IPInfoLabel = ui.HeaderComponent.IPInfoLabel
	ui.RoomNameLabel = ui.HeaderComponent.RoomNameLabel
}

// setupTabs configures tabs using components
func (ui *UIManager) setupTabs() {
	// Create tabs container with fixed size
	ui.TabContainer = container.NewAppTabs(
		container.NewTabItem("Home", ui.HomeTabComponent.GetContainer()),
		// About and Settings tabs removed as they're no longer used
	)

	// Limit maximum size of tabs container
	ui.TabContainer.SetTabLocation(container.TabLocationTop)
	maxSize := fyne.NewSize(300, 520)
	container.NewWithoutLayout(ui.TabContainer).Resize(maxSize)
}

// showSystemTrayNotification exibe uma notificação de sistema a partir do ícone na bandeja
func (ui *UIManager) showSystemTrayNotification(title, content string) {
	ui.App.SendNotification(&fyne.Notification{
		Title:   title,
		Content: content,
	})
}

// Start launches the graphical interface
func (ui *UIManager) Start() {
	// Pre-initialization of essential components
	ui.preInitializeComponents()

	// Configure main menu
	ui.setupMainMenu()

	// Configura ícone na bandeja do sistema
	ui.setupSystemTray()

	// Configure main interface
	ui.setupMainInterface()

	// Modifica o comportamento de fechar a janela para minimizar para a bandeja em vez de sair
	ui.MainWindow.SetCloseIntercept(func() {
		// Apenas esconde a janela em vez de fechar completamente
		ui.MainWindow.Hide()
	})

	// Show main window
	ui.MainWindow.ShowAndRun()
}

// preInitializeComponents pre-initializes essential interface components
func (ui *UIManager) preInitializeComponents() {
	// Ensure NetworkManager is ready before proceeding
	if ui.VPN == nil || ui.VPN.NetworkManager == nil {
		log.Println("Warning: VPN or NetworkManager not properly initialized")
		return
	}

	// No need to initialize room dialog here - it will be created when needed
}

// setupSystemTray configura o ícone na bandeja do sistema e seu menu de contexto
func (ui *UIManager) setupSystemTray() {
	// Verifica se o aplicativo suporta ícones na bandeja
	if deskApp, ok := ui.App.(desktop.App); ok {
		// Configura o ícone personalizado para a bandeja do sistema
		deskApp.SetSystemTrayIcon(icon.VPN)

		// Show initial notification
		ui.App.SendNotification(&fyne.Notification{
			Title:   "goVPN",
			Content: "Application is running",
		})

		// Configura menu para o ícone na bandeja
		deskApp.SetSystemTrayMenu(fyne.NewMenu("",
			fyne.NewMenuItem("Open goVPN", func() {
				ui.MainWindow.Show()
			}),
			fyne.NewMenuItemSeparator(), // Separador para melhorar a organização
			fyne.NewMenuItem("Connect to Network", func() {
				ui.MainWindow.Show() // Mostra a janela principal
				if ui.ConnectDialog == nil {
					ui.ConnectDialog = NewConnectDialog(ui)
				}
				ui.ConnectDialog.Show()
			}),
			fyne.NewMenuItem("Disconnect", func() {
				if ui.VPN.IsConnected && ui.VPN.NetworkManager != nil {
					err := ui.VPN.NetworkManager.LeaveRoom()
					if err != nil {
						log.Printf("Error disconnecting: %v", err)
					}
					ui.VPN.IsConnected = false
					ui.updatePowerButtonState()
					ui.refreshNetworkList()

					// Notifica o usuário sobre a desconexão
					ui.showSystemTrayNotification("goVPN", "Disconnected from VPN network")
				}
			}),
			fyne.NewMenuItemSeparator(), // Outro separador
			fyne.NewMenuItem("Quit", func() {
				// Se estiver conectado, desconecta antes de fechar
				if ui.VPN.IsConnected && ui.VPN.NetworkManager != nil {
					ui.VPN.NetworkManager.LeaveRoom()
				}

				// // Força o encerramento da aplicação usando os
				// ui.App.Quit()

				// Força o encerramento do programa se o Quit não funcionar
				log.Println("Encerrando a aplicação...")
				os.Exit(0)
			}),
		))

		log.Println("System tray icon initialized successfully")
	} else {
		log.Println("System tray is not supported on this platform")
	}
}

// createWindow creates and configures a new window with specified parameters
func (ui *UIManager) createWindow(title string, width, height int, resizable bool) fyne.Window {
	window := ui.App.NewWindow(title)
	window.Resize(fyne.NewSize(float32(width), float32(height)))
	window.SetFixedSize(!resizable)

	// We only ensure the window is displayed above others, but doesn't affect the application lifecycle
	window.SetOnClosed(func() {
		// Window closed, but we don't terminate the application
		// The window will be recreated the next time it's needed
	})

	return window
}

// ShowMessage displays a simple message dialog to the user
func (ui *UIManager) ShowMessage(title, message string) {
	dialog.ShowInformation(title, message, ui.MainWindow)
}

// UpdateThemeColors atualiza as cores de elementos que dependem do tema
func (ui *UIManager) UpdateThemeColors() {
	// Atualiza a cor do texto do IPv4 no HeaderComponent
	if ui.HeaderComponent != nil && ui.HeaderComponent.IPInfoLabel != nil {
		// Verifica o tema atual
		isDark := ui.App.Settings().ThemeVariant() == theme.VariantDark

		// Define a cor apropriada dependendo do tema
		if isDark {
			ui.HeaderComponent.IPInfoLabel.Color = color.NRGBA{R: 255, G: 255, B: 0, A: 255} // Amarelo para tema escuro
		} else {
			ui.HeaderComponent.IPInfoLabel.Color = color.NRGBA{R: 0, G: 60, B: 120, A: 255} // Azul para tema claro
		}

		ui.HeaderComponent.IPInfoLabel.Refresh()
	}
}
