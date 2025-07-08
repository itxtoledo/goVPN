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

// JoinWindow manages the room joining interface as a window
type JoinWindow struct {
	App        fyne.App
	MainWindow fyne.Window
	JoinRoom   func(string, string, string) (*models.JoinRoomResponse, error)
	SaveRoom   func(string, string, string) error
	Username   string
	Window     fyne.Window
	isOpen     bool
}

// NewJoinWindow creates a new room joining window
func NewJoinWindow(
	app fyne.App,
	mainWindow fyne.Window,
	joinRoom func(string, string, string) (*models.JoinRoomResponse, error),
	saveRoom func(string, string, string) error,
	username string,
) *JoinWindow {
	return &JoinWindow{
		App:        app,
		MainWindow: mainWindow,
		JoinRoom:   joinRoom,
		SaveRoom:   saveRoom,
		Username:   username,
	}
}

// Show displays the room joining interface as a new window
func (jw *JoinWindow) Show(validatePassword func(string) bool, configurePasswordEntry func(*widget.Entry)) {
	// Create a new window using the existing app instance
	jw.Window = jw.App.NewWindow("Join Network")
	jw.Window.Resize(fyne.NewSize(320, 260))
	jw.Window.SetFixedSize(true)
	jw.Window.CenterOnScreen()

	// Mark window as open
	jw.isOpen = true

	// Set close callback to reset the global instance when window closes
	jw.Window.SetCloseIntercept(func() {
		jw.isOpen = false
		globalJoinWindow = nil
		jw.Window.Close()
	})

	// Create title with icon
	titleIcon := widget.NewIcon(theme.ComputerIcon())
	titleLabel := widget.NewLabel("Join Network")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleContainer := container.NewHBox(titleIcon, titleLabel)

	// Create form inputs with better styling
	roomIDEntry := widget.NewEntry()
	roomIDEntry.PlaceHolder = "Room ID (e.g. ABC123)"

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "4-digit PIN"
	configurePasswordEntry(passwordEntry)

	// Add keyboard shortcuts
	roomIDEntry.OnSubmitted = func(text string) {
		passwordEntry.FocusGained()
	}

	passwordEntry.OnSubmitted = func(text string) {
		// Trigger join button when Enter is pressed on password field
		if roomIDEntry.Text != "" && validatePassword(text) {
			// Will be triggered by the join button logic
		}
	}

	// Create compact form with better spacing
	formContainer := container.NewVBox(
		widget.NewLabel("Room ID:"),
		container.NewPadded(roomIDEntry),
		widget.NewLabel("Password:"),
		container.NewPadded(passwordEntry),
	)

	// Create buttons
	var joinButton *widget.Button
	joinButton = widget.NewButtonWithIcon("Join Network", theme.ConfirmIcon(), func() {
		roomID := roomIDEntry.Text
		password := passwordEntry.Text

		if roomID == "" {
			dialog.ShowError(errors.New("room ID cannot be empty"), jw.Window)
			return
		}

		if !validatePassword(password) {
			dialog.ShowError(errors.New("password must be exactly 4 digits"), jw.Window)
			return
		}

		// Show loading indicator
		joinButton.SetText("Joining...")
		joinButton.Disable()

		go func() {
			_, err := jw.JoinRoom(roomID, password, jw.Username)

			// Use goroutine to update UI
			go func() {
				joinButton.SetText("Join Network")
				joinButton.Enable()

				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to join network: %v", err), jw.Window)
					return
				}

				dialog.ShowInformation("Success!", "Successfully joined the network!", jw.Window)
				jw.isOpen = false
				globalJoinWindow = nil
				jw.Window.Close()
			}()
		}()
	})

	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		jw.isOpen = false
		globalJoinWindow = nil
		jw.Window.Close()
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

	jw.Window.SetContent(content)
	jw.Window.Show()

	// Set focus on the room ID field when window opens
	jw.Window.Canvas().Focus(roomIDEntry)
}
