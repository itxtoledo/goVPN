// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/header_component.go
package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// HeaderComponent represents the application header
type HeaderComponent struct {
	UI            *UIManager
	PowerButton   *widget.Button
	IPInfoLabel   *widget.Label
	RoomNameLabel *widget.Label
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

// CreateHeaderContainer creates the header container
func (h *HeaderComponent) CreateHeaderContainer() *fyne.Container {
	// Defining a fixed height for the header
	headerHeight := 60.0
	maxWidth := 300.0

	// Container for the power button centered vertically
	powerContainer := container.New(layout.NewCenterLayout(), h.PowerButton)
	powerContainer.Resize(fyne.NewSize(40, float32(headerHeight)))

	// Container for the IP information centered vertically
	// Reducing width to ensure it doesn't exceed the limit
	ipContainer := container.New(layout.NewCenterLayout(), h.IPInfoLabel)
	ipContainer.Resize(fyne.NewSize(180, float32(headerHeight)))

	// Configure IP layout to ensure text is displayed correctly
	h.IPInfoLabel.Wrapping = fyne.TextTruncate
	h.IPInfoLabel.Resize(fyne.NewSize(170, 30))

	// Main header container with horizontal layout and padding
	headerTop := container.New(
		layout.NewHBoxLayout(),
		powerContainer,
		layout.NewSpacer(),
		ipContainer,
	)
	headerTop.Resize(fyne.NewSize(float32(maxWidth), float32(headerHeight)))

	// Room name label with controlled size
	h.RoomNameLabel.Wrapping = fyne.TextTruncate
	h.RoomNameLabel.Resize(fyne.NewSize(280, 20))

	roomNameContainer := container.NewHBox(
		layout.NewSpacer(),
		h.RoomNameLabel,
	)
	roomNameContainer.Resize(fyne.NewSize(float32(maxWidth), 20))

	// Complete header container with fixed size
	header := container.NewVBox(
		headerTop,
		roomNameContainer,
	)
	header.Resize(fyne.NewSize(float32(maxWidth), float32(headerHeight+20)))

	return container.NewMax(header)
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
