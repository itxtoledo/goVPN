// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/home_tab_component.go
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// HomeTabComponent represents the content of the Home tab
type HomeTabComponent struct {
	UI           *UIManager
	Container    *fyne.Container
	NetworkTree  *NetworkTreeComponent
	EmptyContent *fyne.Container
}

// NewHomeTabComponent creates a new instance of the Home tab component
func NewHomeTabComponent(ui *UIManager, networkTree *NetworkTreeComponent) *HomeTabComponent {
	comp := &HomeTabComponent{
		UI:          ui,
		NetworkTree: networkTree,
		Container:   container.NewMax(),
	}

	// Define maximum size for the container
	comp.Container.Resize(fyne.NewSize(280, 500)) // A bit smaller than the window size

	comp.createEmptyContent()
	comp.updateContent()

	return comp
}

// createEmptyContent creates the content to be displayed when no networks are available
func (h *HomeTabComponent) createEmptyContent() {
	// Creating buttons for when there are no networks
	createNetButton := widget.NewButton("Create a Network", func() {
		if h.UI.RoomDialog == nil {
			h.UI.RoomDialog = NewRoomDialog(h.UI)
		}
		h.UI.RoomDialog.Show()
	})
	createNetButton.Importance = widget.HighImportance

	connectNetButton := widget.NewButton("Connect to a Network", func() {
		if h.UI.ConnectDialog == nil {
			h.UI.ConnectDialog = NewConnectDialog(h.UI)
		}
		h.UI.ConnectDialog.Show()
	})

	// Limits the maximum size of buttons
	createNetButton.Resize(fyne.NewSize(260, 40))
	connectNetButton.Resize(fyne.NewSize(260, 40))

	// Creating an informative text with dynamic online/offline status
	infoText := "This area will list your networks and peers. You are now " +
		func() string {
			if h.UI.VPN.IsConnected {
				return "online"
			}
			return "offline"
		}() +
		", but this computer is not yet a member in a goVPN network."

	// Informative text with centered alignment
	statusText := widget.NewLabelWithStyle(
		infoText,
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	// Limits the size of texts to avoid unintentional line breaking
	statusText.Wrapping = fyne.TextWrapWord
	statusText.Resize(fyne.NewSize(260, 80))

	// Centering all texts horizontally
	title := widget.NewLabelWithStyle("No networks available", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	title.Resize(fyne.NewSize(260, 30))

	description := widget.NewLabelWithStyle("You are not connected to any network. Choose an option:",
		fyne.TextAlignCenter, fyne.TextStyle{})
	description.Resize(fyne.NewSize(260, 40))
	description.Wrapping = fyne.TextWrapWord

	// More compact button container
	buttonContainer := container.NewVBox(
		createNetButton,
		connectNetButton,
	)

	// Main content with controlled size
	h.EmptyContent = container.NewVBox(
		title,
		widget.NewSeparator(),
		description,
		buttonContainer,
		widget.NewSeparator(),
		statusText,
	)

	// Define a fixed size for the empty container
	h.EmptyContent.Resize(fyne.NewSize(280, 400))
}

// updateContent updates the content of the Home tab based on the connection status
func (h *HomeTabComponent) updateContent() {
	h.Container.Objects = nil // Remove all objects

	if len(h.UI.NetworkUsers) == 0 && !h.UI.VPN.IsConnected {
		// Using a simpler layout to avoid unwanted window expansion
		h.Container.Add(container.NewCenter(h.EmptyContent))
	} else {
		// Display the network tree when there are networks or is connected
		h.Container.Add(h.NetworkTree.GetNetworkTree())
	}

	h.Container.Refresh()
}

// GetContainer returns the Home tab container
func (h *HomeTabComponent) GetContainer() *fyne.Container {
	return h.Container
}
