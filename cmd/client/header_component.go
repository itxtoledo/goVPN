// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/header_component.go
package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/icon"
)

// HeaderComponent representa o componente de cabeçalho da aplicação
type HeaderComponent struct {
	UI                  *UIManager
	PowerButton         *widget.Button
	UserIPLabel         *widget.Label
	UsernameLabel       *widget.Label
	RoomLabel           *widget.Label
	BackendStatusLabel  *widget.Label
	BackendActivity     *widget.Activity
	MenuButton          *widget.Button
	defaultWebsocketURL string
}

// NewHeaderComponent cria uma nova instância do componente de cabeçalho
func NewHeaderComponent(ui *UIManager, defaultWebsocketURL string) *HeaderComponent {
	hc := &HeaderComponent{
		UI: ui,
	}

	// Criar componentes de UI
	// Use theme.MediaPlayIcon instead of a custom icon to avoid potential nil issues
	hc.PowerButton = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		hc.toggleConnection()
	})
	hc.PowerButton.Importance = widget.HighImportance // Make power button more prominent

	// Criar label para IP do usuário
	hc.UserIPLabel = widget.NewLabelWithData(hc.UI.RealtimeData.UserIP)
	hc.UserIPLabel.TextStyle = fyne.TextStyle{Monospace: true} // Use monospace for IP

	// Criar labels com dados em tempo real vinculados
	hc.UsernameLabel = widget.NewLabelWithData(hc.UI.RealtimeData.Username)
	hc.UsernameLabel.TextStyle = fyne.TextStyle{Bold: true} // Make username more prominent

	hc.RoomLabel = widget.NewLabelWithData(hc.UI.RealtimeData.RoomName)

	// Status do Backend
	hc.BackendStatusLabel = widget.NewLabel("Backend: Disconnected")
	hc.BackendStatusLabel.TextStyle = fyne.TextStyle{Monospace: true} // Use monospace for status

	// Activity widget para indicar conexão ativa
	hc.BackendActivity = widget.NewActivity()
	hc.BackendActivity.Stop() // Inicialmente parado

	// Botão de menu
	hc.MenuButton = widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		hc.showMenu()
	})

	// Configure listeners para atualização automática
	hc.configureListeners()

	return hc
}

// configureListeners configura os listeners para os bindings
func (hc *HeaderComponent) configureListeners() {
	// Listener para o estado da conexão
	hc.UI.RealtimeData.ConnectionState.AddListener(binding.NewDataListener(func() {
		hc.updateBackendStatus()
		hc.updatePowerButtonState()
	}))

	// Listener para o nome da sala
	hc.UI.RealtimeData.RoomName.AddListener(binding.NewDataListener(func() {
		hc.updateRoomName()
	}))

	// Listener para o nome de usuário
	hc.UI.RealtimeData.Username.AddListener(binding.NewDataListener(func() {
		hc.updateUsername()
	}))

	// Listener para o IP do usuário
	hc.UI.RealtimeData.UserIP.AddListener(binding.NewDataListener(func() {
		hc.updateUserIP()
	}))
}

// CreateHeaderContainer cria o container do cabeçalho
func (hc *HeaderComponent) CreateHeaderContainer() *fyne.Container {
	// Container para o status do backend (compacto)
	backendStatusContainer := container.NewHBox(
		hc.BackendStatusLabel,
		hc.BackendActivity,
	)

	// Container para informações do usuário (IP e nome)
	userInfoContainer := container.NewVBox(
		hc.UserIPLabel,
		hc.UsernameLabel,
	)

	// Container superior com botão de energia, informações do usuário e botão de menu
	topContainer := container.NewBorder(
		nil, nil,
		container.NewHBox(hc.PowerButton, userInfoContainer),
		hc.MenuButton,
		backendStatusContainer,
	)

	// Container inferior com informações de sala (compacto)
	bottomContainer := container.NewHBox(
		widget.NewIcon(theme.HomeIcon()),
		hc.RoomLabel,
	)

	// Container principal mais compacto
	headerContainer := container.NewVBox(
		topContainer,
		bottomContainer,
		widget.NewSeparator(),
	)

	return headerContainer
}

// toggleConnection alterna o estado da conexão
func (hc *HeaderComponent) toggleConnection() {
	state, _ := hc.UI.RealtimeData.ConnectionState.Get()
	connectionState := data.ConnectionState(state)

	if connectionState == data.StateDisconnected {
		// Conectar
		go func() {
			log.Println("Connecting to VPN network...")
			hc.UI.VPN.Run(hc.defaultWebsocketURL, hc.UI.RealtimeData, hc.UI.refreshNetworkList, hc.UI.refreshUI)
		}()
	} else {
		// Desconectar
		go func() {
			log.Println("Disconnecting from VPN network...")
			if hc.UI.VPN.NetworkManager != nil {
				err := hc.UI.VPN.NetworkManager.Disconnect()
				if err != nil {
					log.Printf("Error disconnecting: %v", err)
				}
			}
		}()
	}
}

// updatePowerButtonState atualiza o estado do botão de energia
func (hc *HeaderComponent) updatePowerButtonState() {
	state, _ := hc.UI.RealtimeData.ConnectionState.Get()
	connectionState := data.ConnectionState(state)

	// Atualizar ícone do botão
	hc.PowerButton.SetIcon(icon.Power)
	if connectionState == data.StateDisconnected {
		hc.PowerButton.Importance = widget.HighImportance
	} else {
		hc.PowerButton.Importance = widget.DangerImportance
	}

	// Atualizar o botão
	hc.PowerButton.Refresh()
}

// updateUsername atualiza o nome de usuário
func (hc *HeaderComponent) updateUsername() {
	// O nome de usuário já está vinculado diretamente via binding
	// Esta função é mantida para compatibilidade e possíveis futuras extensões
}

// updateUserIP atualiza o IP do usuário
func (hc *HeaderComponent) updateUserIP() {
	// O IP do usuário já está vinculado diretamente via binding
	// Esta função é mantida para compatibilidade e possíveis futuras extensões
}

// updateRoomName atualiza o nome da sala
func (hc *HeaderComponent) updateRoomName() {
	// O nome da sala já está vinculado diretamente via binding
	// Esta função é mantida para compatibilidade e possíveis futuras extensões
}

// updateBackendStatus atualiza o status do backend
func (hc *HeaderComponent) updateBackendStatus() {
	state, _ := hc.UI.RealtimeData.ConnectionState.Get()
	connectionState := data.ConnectionState(state)

	switch connectionState {
	case data.StateDisconnected:
		hc.BackendStatusLabel.SetText("Backend: Disconnected")
		hc.BackendActivity.Stop() // Para a animação quando desconectado
	case data.StateConnecting:
		hc.BackendStatusLabel.SetText("Backend: Connecting...")
		hc.BackendActivity.Start() // Inicia a animação ao conectar
	case data.StateConnected:
		hc.BackendStatusLabel.SetText("Backend: Connected")
		hc.BackendActivity.Start() // Mantém a animação enquanto conectado
	}

	hc.BackendStatusLabel.Refresh()
}

// showMenu mostra o menu de opções
func (hc *HeaderComponent) showMenu() {
	// Criar um menu popup
	menu := widget.NewPopUpMenu(
		fyne.NewMenu(
			"Options",
			fyne.NewMenuItem("About", func() {
				// Always create a new About window to ensure content is properly initialized
				hc.UI.AboutWindow = NewAboutWindow(hc.UI)
				hc.UI.AboutWindow.Show()
			}),
			fyne.NewMenuItem("Exit", func() {
				hc.UI.MainWindow.Close()
			}),
		),
		hc.UI.MainWindow.Canvas(),
	)

	// Exibir o menu
	menu.ShowAtPosition(fyne.NewPos(
		hc.MenuButton.Position().X,
		hc.MenuButton.Position().Y+hc.MenuButton.Size().Height,
	))
}
