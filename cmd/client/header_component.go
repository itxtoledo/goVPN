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
)

// HeaderComponent represents the application header
type HeaderComponent struct {
	UI             *UIManager
	PowerButton    *widget.Button
	AboutButton    *widget.Button
	SettingsButton *widget.Button
	IPInfoLabel    *widget.Label
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
	h.PowerButton = widget.NewButtonWithIcon("", h.loadPowerButtonResource(), func() {
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

	// Label for IP information
	h.IPInfoLabel = widget.NewLabel("YOUR IPV4")

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
		h.updateRoomName()
		h.UI.refreshNetworkList()
	} else {
		// Connect
		// Make sure the connection window is initialized
		if h.UI.ConnectWindow == nil {
			h.UI.ConnectWindow = NewConnectWindow(h.UI)
		}
		h.UI.ConnectWindow.Show()
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
	// Ensure Settings window is initialized
	if h.UI.SettingsWindow == nil {
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

	// Container for the IP information centered vertically
	// Reducing width to ensure it doesn't exceed the limit
	ipContainer := container.New(layout.NewCenterLayout(), h.IPInfoLabel)
	ipContainer.Resize(fyne.NewSize(180, float32(headerHeight)))

	// Configure IP layout to ensure text is displayed correctly
	h.IPInfoLabel.Wrapping = fyne.TextTruncate
	h.IPInfoLabel.Resize(fyne.NewSize(170, 30))

	// Main header container with horizontal layout and padding
	headerTop := container.New(
		layout.NewPaddedLayout(), // Using padded layout for horizontal spacing
		container.New(
			layout.NewHBoxLayout(),
			powerContainer,
			layout.NewSpacer(),
			ipContainer,
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

	h.IPInfoLabel.SetText(ipv4)
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
		h.StatusLabel.SetText("Initializing...")
		return
	}

	switch h.UI.VPN.NetworkManager.GetConnectionState() {
	case ConnectionStateConnected:
		// Connected - Green
		h.BackendStatus.FillColor = color.RGBA{0, 255, 0, 255}
		h.BackendStatus.StrokeColor = color.RGBA{0, 200, 0, 255}
		h.StatusLabel.SetText("Connected")
	case ConnectionStateConnecting:
		// Connecting - Yellow
		h.BackendStatus.FillColor = color.RGBA{255, 255, 0, 255}
		h.BackendStatus.StrokeColor = color.RGBA{200, 200, 0, 255}
		h.StatusLabel.SetText("Connecting...")
	case ConnectionStateDisconnected:
		// Disconnected - Red
		h.BackendStatus.FillColor = color.RGBA{255, 0, 0, 255}
		h.BackendStatus.StrokeColor = color.RGBA{200, 0, 0, 255}
		h.StatusLabel.SetText("Disconnected")
	}
	h.BackendStatus.Refresh()
	h.StatusLabel.Refresh()
}

// loadPowerButtonResource loads the power button SVG icon
func (h *HeaderComponent) loadPowerButtonResource() fyne.Resource {
	// Load the icon from the SVG file
	res, err := fyne.LoadResourceFromPath("power-button.svg")
	if err != nil {
		log.Printf("Error loading power button icon: %v", err)
		return theme.CancelIcon() // Return a default icon in case of failure
	}
	return res
}
