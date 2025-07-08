package main

import (
	"errors"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// Global variable to ensure only one room window can be open
var globalRoomWindow *RoomWindow

// RoomWindow manages the room (network) creation interface as a window
type RoomWindow struct {
	App        fyne.App
	MainWindow fyne.Window
	CreateRoom func(string, string) (*models.CreateRoomResponse, error)
	SaveRoom   func(string, string, string) error
	GetRoomID  func() string
	Username   string
	Window     fyne.Window
	isOpen     bool
}

// NewRoomWindow creates a new room creation window
func NewRoomWindow(
	app fyne.App,
	mainWindow fyne.Window,
	createRoom func(string, string) (*models.CreateRoomResponse, error),
	saveRoom func(string, string, string) error,
	getRoomID func() string,
	username string,
) *RoomWindow {
	return &RoomWindow{
		App:        app,
		MainWindow: mainWindow,
		CreateRoom: createRoom,
		SaveRoom:   saveRoom,
		GetRoomID:  getRoomID,
		Username:   username,
	}
}

// Show displays the room creation interface as a new window
func (rw *RoomWindow) Show(validatePassword func(string) bool, configurePasswordEntry func(*widget.Entry)) {
	// Create a new window using the existing app instance
	rw.Window = rw.App.NewWindow("Create Network")
	rw.Window.Resize(fyne.NewSize(320, 260))
	rw.Window.SetFixedSize(true)
	rw.Window.CenterOnScreen()

	// Mark window as open
	rw.isOpen = true

	// Set close callback to reset the global instance when window closes
	rw.Window.SetCloseIntercept(func() {
		rw.isOpen = false
		globalRoomWindow = nil
		rw.Window.Close()
	})

	// Create title with icon
	titleIcon := widget.NewIcon(theme.ContentAddIcon())
	titleLabel := widget.NewLabel("Create Network")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleContainer := container.NewHBox(titleIcon, titleLabel)

	// Create form inputs with better styling
	nameEntry := widget.NewEntry()
	nameEntry.PlaceHolder = "Network name"

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "4-digit PIN"
	configurePasswordEntry(passwordEntry)

	// Add keyboard shortcuts
	nameEntry.OnSubmitted = func(text string) {
		passwordEntry.FocusGained()
	}

	passwordEntry.OnSubmitted = func(text string) {
		// Trigger create button when Enter is pressed on password field
		if nameEntry.Text != "" && validatePassword(text) {
			// Will be triggered by the create button logic
		}
	}

	// Create compact form with better spacing
	formContainer := container.NewVBox(
		widget.NewLabel("Network Name:"),
		container.NewPadded(nameEntry),
		widget.NewLabel("Password:"),
		container.NewPadded(passwordEntry),
	)

	// Create buttons
	var createButton *widget.Button
	createButton = widget.NewButtonWithIcon("Create Network", theme.ConfirmIcon(), func() {
		name := nameEntry.Text
		password := passwordEntry.Text

		// Validate name
		if name == "" {
			dialog.ShowError(errors.New("network name cannot be empty"), rw.Window)
			return
		}

		// Validate password using the abstract function
		if !validatePassword(password) {
			dialog.ShowError(errors.New("password must be exactly 4 digits"), rw.Window)
			return
		}

		// Show loading indicator
		createButton.SetText("Creating...")
		createButton.Disable()

		// Create network in a goroutine
		go func() {
			// Send create room command to backend
			res, err := rw.CreateRoom(name, password)

			// Update UI on the main thread
			go func() {
				createButton.SetText("Create Network")
				createButton.Enable()

				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to create network: %v", err), rw.Window)
					return
				}

				// Get room ID created by the server
				roomID := res.RoomID

				if roomID != "" {
					// Save room to database
					err := rw.SaveRoom(roomID, name, password)
					if err != nil {
						log.Printf("Error saving room to database: %v", err)
					}

					// Show success dialog with room ID
					roomIDEntry := widget.NewEntry()
					roomIDEntry.Text = roomID
					roomIDEntry.Disable()

					content := container.NewVBox(
						widget.NewLabel("Network created successfully!"),
						widget.NewLabel(""),
						widget.NewLabel("Network ID (share this with friends):"),
						roomIDEntry,
						widget.NewButton("Copy to Clipboard", func() {
							rw.Window.Clipboard().SetContent(roomID)
							dialog.ShowInformation("Copied", "Network ID copied to clipboard", rw.Window)
						}),
					)

					successDialog := dialog.NewCustom("Success", "Close", content, rw.Window)
					successDialog.Show()

					// Close the create window after showing success
					rw.isOpen = false
					globalRoomWindow = nil
					rw.Window.Close()
				} else {
					// If roomID is empty, it's likely an error occurred but wasn't caught
					dialog.ShowError(errors.New("failed to create network: no network ID returned"), rw.Window)
				}
			}()
		}()
	})

	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		rw.isOpen = false
		globalRoomWindow = nil
		rw.Window.Close()
	})

	// Style buttons
	createButton.Importance = widget.HighImportance

	// Create button container with better spacing
	buttonContainer := container.NewGridWithColumns(2, cancelButton, createButton)

	// Create main content with compact design
	content := container.NewVBox(
		container.NewPadded(titleContainer),
		widget.NewSeparator(),
		container.NewPadded(formContainer),
		widget.NewSeparator(),
		container.NewPadded(buttonContainer),
	)

	rw.Window.SetContent(content)
	rw.Window.Show()

	// Set focus on the name field when window opens
	rw.Window.Canvas().Focus(nameEntry)
}
