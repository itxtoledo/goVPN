package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// LoginWindow manages the login interface
type LoginWindow struct {
	UI            *UIManager
	Window        fyne.Window
	ServerEntry   *widget.Entry
	UsernameEntry *widget.Entry
	PasswordEntry *widget.Entry
	LoginButton   *widget.Button
	Container     *fyne.Container
}

// NewLoginWindow creates a new login window
func NewLoginWindow(ui *UIManager) *LoginWindow {
	loginWindow := &LoginWindow{
		UI:     ui,
		Window: ui.createWindow("Login - goVPN", 400, 300, false),
	}

	return loginWindow
}

// Show displays the login window
func (lw *LoginWindow) Show() {
	// If the window has been closed, recreate it
	if lw.Window == nil {
		lw.Window = lw.UI.createWindow("Login - goVPN", 400, 300, false)
	}

	// Initialize the necessary components before showing the window
	content := lw.CreateContent()

	// Set the window content
	lw.Window.SetContent(content)

	// Display the window centered
	lw.Window.CenterOnScreen()
	lw.Window.Show()
}

// CreateContent creates the content for the login window
func (lw *LoginWindow) CreateContent() fyne.CanvasObject {
	// Input fields
	lw.ServerEntry = widget.NewEntry()
	lw.ServerEntry.SetPlaceHolder("Signaling server address")
	lw.ServerEntry.SetText(lw.UI.VPN.NetworkManager.SignalServer)

	lw.UsernameEntry = widget.NewEntry()
	lw.UsernameEntry.SetPlaceHolder("Username")

	lw.PasswordEntry = widget.NewPasswordEntry()
	lw.PasswordEntry.SetPlaceHolder("Password")

	// Login button
	lw.LoginButton = widget.NewButton("Connect", func() {
		lw.login()
	})

	// Login form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Server", Widget: lw.ServerEntry},
			{Text: "Username", Widget: lw.UsernameEntry},
			{Text: "Password", Widget: lw.PasswordEntry},
		},
		SubmitText: "Connect",
		OnSubmit:   lw.login,
	}

	// Main container
	lw.Container = container.NewVBox(
		widget.NewLabel("Login to goVPN Server"),
		form,
		container.NewHBox(
			widget.NewButton("Cancel", func() {
				lw.Window.Hide()
			}),
		),
	)

	return lw.Container
}

// login processes the login attempt
func (lw *LoginWindow) login() {
	// Update the signaling server address
	lw.UI.VPN.NetworkManager.SignalServer = lw.ServerEntry.Text

	// Here the real authentication logic would be implemented
	// For this example, we just try to connect to the server
	err := lw.UI.VPN.NetworkManager.Connect()
	if err != nil {
		lw.UI.ShowMessage("Error", "Could not connect to the server: "+err.Error())
		return
	}

	// Hide the login window after success
	lw.Window.Hide()
}
