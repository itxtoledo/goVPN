package main

import (
	"fmt"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ConnectWindow manages the VPN connection interface
type ConnectWindow struct {
	UI            *UIManager
	Window        fyne.Window
	RoomIDEntry   *widget.Entry
	PasswordEntry *widget.Entry
	ConnectButton *widget.Button
	Container     *fyne.Container
	Status        *widget.Label
}

// NewConnectWindow creates a new connection window
func NewConnectWindow(ui *UIManager) *ConnectWindow {
	connectWindow := &ConnectWindow{
		UI:     ui,
		Window: ui.createWindow("Connect to VPN", 400, 300, false),
	}

	// Add handler for when the window is closed
	connectWindow.Window.SetOnClosed(func() {
		connectWindow.Window = nil
	})

	return connectWindow
}

// Show displays the connection window
func (cw *ConnectWindow) Show() {
	// If the window has been closed, recreate it
	if cw.Window == nil {
		cw.Window = cw.UI.createWindow("Connect to VPN", 400, 300, false)
		// Re-add the handler for when the window is closed
		cw.Window.SetOnClosed(func() {
			cw.Window = nil
		})
	}

	// Ensure content is created before displaying the window
	content := cw.CreateContent()

	// Set the window content
	cw.Window.SetContent(content)

	// Clear the fields
	cw.RoomIDEntry.SetText("")
	cw.PasswordEntry.SetText("")
	cw.Status.SetText("")

	// Display the window centered
	cw.Window.CenterOnScreen()
	cw.Window.Show()
}

// CreateContent creates the content for the connection window
func (cw *ConnectWindow) CreateContent() fyne.CanvasObject {
	// Input fields
	cw.RoomIDEntry = widget.NewEntry()
	cw.RoomIDEntry.SetPlaceHolder("Room ID")

	cw.PasswordEntry = widget.NewPasswordEntry()
	cw.PasswordEntry.SetPlaceHolder("Room Password")

	// Restrict to only numbers and maximum of 4 characters
	cw.PasswordEntry.Validator = func(s string) error {
		if len(s) > 4 {
			return fmt.Errorf("password must be at most 4 characters")
		}
		for _, r := range s {
			if !unicode.IsDigit(r) {
				return fmt.Errorf("password must contain only numbers")
			}
		}
		return nil
	}

	// Limit input to 4 characters in real-time
	cw.PasswordEntry.OnChanged = func(s string) {
		if len(s) > 4 {
			cw.PasswordEntry.SetText(s[:4])
		}
	}

	// Connection status
	cw.Status = widget.NewLabel("")

	// Action buttons
	cw.ConnectButton = widget.NewButton("Connect", func() {
		cw.connect()
	})

	// Connection form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Room ID", Widget: cw.RoomIDEntry},
			{Text: "Password", Widget: cw.PasswordEntry},
		},
		SubmitText: "Connect",
		OnSubmit: func() {
			cw.connect()
		},
	}

	// Main container
	cw.Container = container.NewVBox(
		widget.NewLabel("Connect to a VPN Room"),
		form,
		cw.Status,
		container.NewHBox(
			widget.NewButton("Cancel", func() {
				cw.Window.Close()
			}),
		),
	)

	return cw.Container
}

// connect attempts to connect to a VPN room
func (cw *ConnectWindow) connect() {
	// Field validation
	if cw.RoomIDEntry.Text == "" || cw.PasswordEntry.Text == "" {
		cw.Status.SetText("Please fill in all fields")
		return
	}

	// Attempt to connect to the signaling server if not already connected
	if !cw.UI.VPN.NetworkManager.IsConnected {
		cw.Status.SetText("Connecting to server...")
		err := cw.UI.VPN.NetworkManager.Connect()
		if err != nil {
			cw.Status.SetText("Connection failed: " + err.Error())
			return
		}
	}

	// Attempt to join the room
	cw.Status.SetText("Joining room...")
	err := cw.UI.VPN.NetworkManager.JoinRoom(cw.RoomIDEntry.Text, cw.PasswordEntry.Text)
	if err != nil {
		cw.Status.SetText("Failed to join room: " + err.Error())
		return
	}

	// Save room information and close the window
	cw.UI.VPN.CurrentRoom = cw.RoomIDEntry.Text
	cw.UI.VPN.NetworkManager.RoomName = cw.RoomIDEntry.Text
	cw.UI.VPN.IsConnected = true

	// Update the toggle button state in the header and connection information
	cw.UI.updatePowerButtonState()
	cw.UI.updateIPInfo()
	cw.UI.updateRoomName()

	cw.Window.Close()
}
