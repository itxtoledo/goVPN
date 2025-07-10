package main

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// Global variable to ensure only one join window can be open
var globalJoinWindow *JoinWindow

// JoinWindow manages the network joining interface as a window
type JoinWindow struct {
	*BaseWindow
	JoinNetwork            func(string, string, string) (*models.JoinNetworkResponse, error)
	ComputerName           string
	ValidatePassword       func(string) bool
	ConfigurePasswordEntry func(*widget.Entry)
	OnNetworkJoined        func(networkID, password string)
}

// NewJoinWindow creates a new network joining window
func NewJoinWindow(
	app fyne.App,
	joinNetwork func(string, string, string) (*models.JoinNetworkResponse, error),
	computername string,
	validatePassword func(string) bool,
	configurePasswordEntry func(*widget.Entry),
	onNetworkJoined func(networkID, password string),
) *JoinWindow {
	jw := &JoinWindow{
		BaseWindow:             NewBaseWindow(app, "Join Network", 320, 260),
		JoinNetwork:            joinNetwork,
		ComputerName:           computername,
		ValidatePassword:       validatePassword,
		ConfigurePasswordEntry: configurePasswordEntry,
		OnNetworkJoined:        onNetworkJoined,
	}

	// Set close callback to reset the global instance when window closes
	jw.BaseWindow.Window.SetOnClosed(func() {
		globalJoinWindow = nil
	})

	return jw
}

func (jw *JoinWindow) Show() {
	// Mark window as open
	globalJoinWindow = jw

	// Create title with icon
	titleIcon := widget.NewIcon(theme.ComputerIcon())
	titleLabel := widget.NewLabel("Join Network")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleContainer := container.NewHBox(titleIcon, titleLabel)

	// Create form inputs with better styling
	networkIDEntry := widget.NewEntry()
	networkIDEntry.PlaceHolder = "Network ID (e.g. ABC123)"

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "4-digit PIN"
	jw.ConfigurePasswordEntry(passwordEntry)

	// Add keyboard shortcuts
	networkIDEntry.OnSubmitted = func(text string) {
		passwordEntry.FocusGained()
	}

	passwordEntry.OnSubmitted = func(text string) {
		// Trigger join button when Enter is pressed on password field
		if networkIDEntry.Text != "" && jw.ValidatePassword(text) {
			// Will be triggered by the join button logic
		}
	}

	// Create compact form with better spacing
	formContainer := container.NewVBox(
		widget.NewLabel("Network ID:"),
		container.NewPadded(networkIDEntry),
		widget.NewLabel("Password:"),
		container.NewPadded(passwordEntry),
	)

	// Create buttons
	var joinButton *widget.Button
	joinButton = widget.NewButtonWithIcon("Join Network", theme.ConfirmIcon(), func() {
		networkID := networkIDEntry.Text
		password := passwordEntry.Text

		if networkID == "" {
			dialog.ShowError(errors.New("network ID cannot be empty"), jw.BaseWindow.Window)
			return
		}

		if !jw.ValidatePassword(password) {
			dialog.ShowError(errors.New("password must be exactly 4 digits"), jw.BaseWindow.Window)
			return
		}

		// Show loading indicator
		joinButton.SetText("Joining...")
		joinButton.Disable()

		go func() {
			_, err := jw.JoinNetwork(networkID, password, jw.ComputerName)

			// Use goroutine to update UI
			go func() {
				joinButton.SetText("Join Network")
				joinButton.Enable()

				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to join network: %v", err), jw.BaseWindow.Window)
					return
				}

				// Invoke the callback with the network details
				jw.OnNetworkJoined(networkID, password)

				// Close the join window after invoking the callback
				jw.BaseWindow.Close()
			}()
		}()
	})

	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		jw.BaseWindow.Close()
	})

	// Style buttons
	joinButton.Importance = widget.HighImportance

	// Create button container with better spacing
	buttonContainer := container.NewGridWithColumns(2, cancelButton, joinButton)

	// Create main content with compact design
	content := container.NewVBox(
		container.NewPadded(titleContainer),
		widget.NewSeparator(),
		container.NewPadded(formContainer),
		widget.NewSeparator(),
		container.NewPadded(buttonContainer),
	)

	jw.BaseWindow.SetContent(content)
	jw.BaseWindow.Show()

	// Set focus on the network ID field when window opens
	jw.BaseWindow.Window.Canvas().Focus(networkIDEntry)
}
