// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/header_component.go
package main

import (
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/icon"
)

// HeaderComponent represents the application header
type HeaderComponent struct {
	UI             *UIManager
	PowerButton    *widget.Button
	AboutButton    *widget.Button
	SettingsButton *widget.Button
	IPInfoLabel    *canvas.Text // Alterado de widget.Label para canvas.Text para controle de cor
	UsernameLabel  *canvas.Text // Label para exibir o nome de usuário
	RoomNameLabel  *widget.Label
	BackendStatus  *canvas.Rectangle // Using rectangle instead of circle for better visibility
	StatusLabel    *widget.Label     // Text label showing connection state
}

// NewHeaderComponent creates a new instance of the header component
func NewHeaderComponent(ui *UIManager) *HeaderComponent {
	header := &HeaderComponent{
		UI: ui,
	}

	header.init()
	return header
}

// init initializes the header components
func (h *HeaderComponent) init() {
	// Create power button
	h.PowerButton = widget.NewButtonWithIcon("", icon.Power, func() {
		h.handlePowerButtonClick()
	})
	h.PowerButton.Importance = widget.DangerImportance

	// Create About button
	h.AboutButton = widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
		h.handleAboutButtonClick()
	})
	h.AboutButton.Importance = widget.LowImportance

	// Create Settings button
	h.SettingsButton = widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		h.handleSettingsButtonClick()
	})
	h.SettingsButton.Importance = widget.LowImportance

	// Create backend connection status indicator (using rectangle for better visibility)
	h.BackendStatus = &canvas.Rectangle{
		FillColor:   color.RGBA{255, 0, 0, 255}, // Red for disconnected
		StrokeColor: color.RGBA{200, 0, 0, 255},
		StrokeWidth: 1,
	}

	// Create status label with smaller text
	h.StatusLabel = widget.NewLabel("Disconnected")
	h.StatusLabel.TextStyle = fyne.TextStyle{Bold: false} // Remove bold for smaller appearance
	h.StatusLabel.Alignment = fyne.TextAlignLeading
	h.StatusLabel.TextStyle.Monospace = true // Using monospace for compact text

	h.updateBackendStatus() // Initialize status

	// Determinar se estamos usando tema escuro
	isDark := false
	if h.UI.App != nil && h.UI.App.Settings() != nil {
		isDark = h.UI.App.Settings().ThemeVariant() == theme.VariantDark
	}

	// Definir a cor do texto com base no tema
	ipTextColor := color.NRGBA{R: 0, G: 60, B: 120, A: 255} // Azul escuro para tema claro
	if isDark {
		ipTextColor = color.NRGBA{R: 255, G: 255, B: 0, A: 255} // Amarelo brilhante para tema escuro
	}

	// Label for IP information usando a cor apropriada
	h.IPInfoLabel = canvas.NewText("YOUR IPV4", ipTextColor)
	h.IPInfoLabel.TextSize = 16
	h.IPInfoLabel.Alignment = fyne.TextAlignCenter

	// Label para nome de usuário (usando cor mais suave)
	usernameColor := color.NRGBA{R: 0, G: 100, B: 100, A: 255} // Verde-azulado para tema claro
	if isDark {
		usernameColor = color.NRGBA{R: 100, G: 255, B: 100, A: 255} // Verde claro para tema escuro
	}
	h.UsernameLabel = canvas.NewText("Not connected", usernameColor)
	h.UsernameLabel.TextSize = 12 // Tamanho menor que o IP
	h.UsernameLabel.Alignment = fyne.TextAlignCenter

	// Label for room name
	h.RoomNameLabel = widget.NewLabel("Room: Not connected")
	h.RoomNameLabel.Alignment = fyne.TextAlignTrailing
}

// handlePowerButtonClick handles the click on the power button
func (h *HeaderComponent) handlePowerButtonClick() {
	if h.UI.VPN.IsConnected {
		// Disconnect
		err := h.UI.VPN.NetworkManager.LeaveRoom()
		if err != nil {
			log.Printf("Error while disconnecting: %v", err)
		}
		h.UI.VPN.IsConnected = false
		h.updatePowerButtonState()
		h.updateIPInfo()
		h.updateUsername() // Atualiza o nome de usuário
		h.updateRoomName()
		h.UI.refreshNetworkList()
	} else {
		// Connect - usando o novo ConnectDialog
		if h.UI.ConnectDialog == nil {
			h.UI.ConnectDialog = NewConnectDialog(h.UI)
		}
		h.UI.ConnectDialog.Show()
	}
}

// handleAboutButtonClick handles the click on the About button
func (h *HeaderComponent) handleAboutButtonClick() {
	// Ensure About window is initialized
	if h.UI.AboutWindow == nil {
		h.UI.AboutWindow = NewAboutWindow(h.UI)
	}
	h.UI.AboutWindow.Show()
}

// handleSettingsButtonClick handles the click on the Settings button
func (h *HeaderComponent) handleSettingsButtonClick() {
	// Always create a new Settings window or use the existing one if not null
	// This ensures we always have a fresh instance after closing
	if h.UI.SettingsWindow == nil || h.UI.SettingsWindow.Window == nil {
		h.UI.SettingsWindow = NewSettingsWindow(h.UI)
	}
	h.UI.SettingsWindow.Show()
}

// CreateHeaderContainer creates the header container
func (h *HeaderComponent) CreateHeaderContainer() *fyne.Container {
	// Defining a fixed height for the header
	headerHeight := 60.0
	maxWidth := 300.0

	// Container for the power button centered vertically with left margin
	powerContainer := container.New(layout.NewCenterLayout(), h.PowerButton)
	powerContainer.Resize(fyne.NewSize(40, float32(headerHeight)))

	// Configure backend status indicator with better visibility
	h.BackendStatus.Resize(fyne.NewSize(10, 10)) // Small rectangle
	h.updateBackendStatus()                      // Update colors based on current state

	// Create a container for the status indicator with label in horizontal layout
	statusContainer := container.NewHBox(
		h.BackendStatus,      // The rectangle comes first in a horizontal layout
		widget.NewLabel(" "), // Small space between rectangle and text
		h.StatusLabel,        // Then the text
	)
	statusContainer.Resize(fyne.NewSize(100, 20)) // Smaller width to prevent taking too much space

	// Container for buttons at the top
	buttonBar := container.NewHBox(
		statusContainer, // Status indicator with text at the left side
		layout.NewSpacer(),
		h.AboutButton,
		h.SettingsButton,
	)
	buttonBar.Resize(fyne.NewSize(float32(maxWidth-40), 30)) // Adjusted for padding

	// Atualize as propriedades do IPInfoLabel para garantir visibilidade e tamanho adequados
	h.IPInfoLabel.TextSize = 16
	h.IPInfoLabel.Alignment = fyne.TextAlignCenter

	// Certifique-se de atualizar o nome de usuário antes de exibi-lo
	h.updateUsername()

	// Container para IP e nome de usuário em layout vertical
	ipUsernameContainer := container.NewVBox(
		h.IPInfoLabel,
		h.UsernameLabel,
	)

	// Container para centralizar o conjunto vertical de IP e nome de usuário
	centerContainer := container.New(layout.NewCenterLayout(), ipUsernameContainer)
	centerContainer.Resize(fyne.NewSize(180, float32(headerHeight)))

	// Main header container with horizontal layout and padding
	headerTop := container.New(
		layout.NewPaddedLayout(), // Using padded layout for horizontal spacing
		container.New(
			layout.NewHBoxLayout(),
			powerContainer,
			layout.NewSpacer(),
			centerContainer,
		),
	)
	headerTop.Resize(fyne.NewSize(float32(maxWidth-40), float32(headerHeight))) // Adjusted for padding

	// Room name label with controlled size
	h.RoomNameLabel.Wrapping = fyne.TextTruncate
	h.RoomNameLabel.Resize(fyne.NewSize(260, 20)) // Adjusted for padding

	roomNameContainer := container.NewHBox(
		layout.NewSpacer(),
		h.RoomNameLabel,
	)
	roomNameContainer.Resize(fyne.NewSize(float32(maxWidth-40), 20)) // Adjusted for padding

	// Complete header container with fixed size
	innerHeader := container.NewVBox(
		buttonBar,
		headerTop,
		roomNameContainer,
	)
	innerHeader.Resize(fyne.NewSize(float32(maxWidth-40), float32(headerHeight+50))) // Adjusted for padding

	// Create padding container with 20px padding on all sides
	paddedContainer := container.NewPadded(innerHeader)
	paddedContainer.Resize(fyne.NewSize(float32(maxWidth), float32(headerHeight+50+40))) // Added padding height

	return container.NewMax(paddedContainer)
}

// updatePowerButtonState updates the visual state of the power button
func (h *HeaderComponent) updatePowerButtonState() {
	if h.UI.VPN.IsConnected {
		h.PowerButton.Importance = widget.HighImportance // Green for connected
	} else {
		h.PowerButton.Importance = widget.DangerImportance // Red for disconnected
	}
	h.PowerButton.Refresh()
}

// updateIPInfo updates the displayed IP information
func (h *HeaderComponent) updateIPInfo() {
	ipv4 := "YOUR IPV4"

	if h.UI.VPN.IsConnected && h.UI.VPN.NetworkManager.VirtualNetwork != nil {
		ipv4 = h.UI.VPN.NetworkManager.VirtualNetwork.GetLocalIP()
	}

	// Configurar cor do texto baseado no tema atual
	isDark := h.UI.App.Settings().ThemeVariant() == theme.VariantDark

	// Para tema escuro, usar cor amarela brilhante para melhor contraste
	// Para tema claro, usar cor azul escuro para melhor contraste
	if isDark {
		h.IPInfoLabel.Color = color.NRGBA{R: 255, G: 255, B: 0, A: 255} // Amarelo brilhante
	} else {
		h.IPInfoLabel.Color = color.NRGBA{R: 0, G: 60, B: 120, A: 255} // Azul escuro
	}

	// Definir tamanho e alinhamento do texto
	h.IPInfoLabel.TextSize = 16
	h.IPInfoLabel.Alignment = fyne.TextAlignCenter
	h.IPInfoLabel.Text = ipv4
	h.IPInfoLabel.Refresh()
}

// updateRoomName updates the displayed room name
func (h *HeaderComponent) updateRoomName() {
	roomName := "Not connected"

	if h.UI.VPN.IsConnected && h.UI.VPN.NetworkManager.RoomName != "" {
		roomName = h.UI.VPN.NetworkManager.RoomName
	}

	h.RoomNameLabel.SetText("Room: " + roomName)
	h.RoomNameLabel.Refresh()
}

// updateBackendStatus updates the backend connection status indicator
func (h *HeaderComponent) updateBackendStatus() {
	// Get connection state from network manager
	if h.UI.VPN.NetworkManager == nil {
		// Network manager not initialized yet
		h.BackendStatus.FillColor = color.RGBA{128, 128, 128, 255} // Gray for unknown state
		h.BackendStatus.StrokeColor = color.RGBA{128, 128, 128, 255}
		h.StatusLabel.SetText("Init...")
		return
	}

	switch h.UI.VPN.NetworkManager.GetConnectionState() {
	case ConnectionStateConnected:
		// Connected - Green
		h.BackendStatus.FillColor = color.RGBA{0, 255, 0, 255}
		h.BackendStatus.StrokeColor = color.RGBA{0, 200, 0, 255}
		h.StatusLabel.SetText("On")
	case ConnectionStateConnecting:
		// Connecting - Yellow
		h.BackendStatus.FillColor = color.RGBA{255, 255, 0, 255}
		h.BackendStatus.StrokeColor = color.RGBA{200, 200, 0, 255}
		h.StatusLabel.SetText("Connecting...")
	case ConnectionStateDisconnected:
		// Disconnected - Red
		h.BackendStatus.FillColor = color.RGBA{255, 0, 0, 255}
		h.BackendStatus.StrokeColor = color.RGBA{200, 0, 0, 255}
		h.StatusLabel.SetText("Off")
	}
	h.BackendStatus.Refresh()
	h.StatusLabel.Refresh()
}

// updateUsername atualiza o nome de usuário exibido
func (h *HeaderComponent) updateUsername() {
	// Obter o nome de usuário das configurações
	config := h.UI.ConfigManager.GetConfig()
	username := config.Username

	if username == "" {
		username = "Anonymous User"
	}

	// Se não estiver conectado, o campo ainda é exibido, mas pode ter uma aparência diferente
	isConnected := h.UI.VPN.IsConnected && h.UI.VPN.NetworkManager != nil

	// Configurar cor do texto baseado no tema atual
	isDark := h.UI.App.Settings().ThemeVariant() == theme.VariantDark

	// Definir cores para o nome de usuário - cores mais vibrantes para conectado, cores mais suaves para desconectado
	if isDark {
		if isConnected {
			h.UsernameLabel.Color = color.NRGBA{R: 100, G: 255, B: 100, A: 255} // Verde claro para tema escuro
		} else {
			h.UsernameLabel.Color = color.NRGBA{R: 70, G: 130, B: 70, A: 255} // Verde escuro esmaecido para tema escuro
		}
	} else {
		if isConnected {
			h.UsernameLabel.Color = color.NRGBA{R: 0, G: 100, B: 100, A: 255} // Verde-azulado para tema claro
		} else {
			h.UsernameLabel.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255} // Cinza para tema claro
		}
	}

	// Definir tamanho e alinhamento do texto
	h.UsernameLabel.TextSize = 12
	h.UsernameLabel.Alignment = fyne.TextAlignCenter
	h.UsernameLabel.Text = username
	h.UsernameLabel.Refresh()
}
