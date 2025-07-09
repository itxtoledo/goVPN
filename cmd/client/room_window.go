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

// Global variable to ensure only one network window can be open
var globalNetworkWindow *NetworkWindow

// NetworkWindow manages the network (network) creation interface as a window
type NetworkWindow struct {
	*BaseWindow
	CreateNetwork          func(string, string) (*models.CreateNetworkResponse, error)
	SaveNetwork            func(string, string, string) error
	GetNetworkID           func() string
	ComputerName           string
	ValidatePassword       func(string) bool
	ConfigurePasswordEntry func(*widget.Entry)
}

// NewNetworkWindow creates a new network creation window
func NewNetworkWindow(
	ui *UIManager,
	createNetwork func(string, string) (*models.CreateNetworkResponse, error),
	saveNetwork func(string, string, string) error,
	getNetworkID func() string,
	computername string,
	validatePassword func(string) bool,
	configurePasswordEntry func(*widget.Entry),
) *NetworkWindow {
	rw := &NetworkWindow{
		BaseWindow:             NewBaseWindow(ui.createWindow, "Create Network", 320, 260),
		CreateNetwork:          createNetwork,
		SaveNetwork:            saveNetwork,
		GetNetworkID:           getNetworkID,
		ComputerName:           computername,
		ValidatePassword:       validatePassword,
		ConfigurePasswordEntry: configurePasswordEntry,
	}

	// Set close callback to reset the global instance when window closes
	// This ensures that only one network window can be open at a time
	rw.BaseWindow.Window.SetOnClosed(func() {
		globalNetworkWindow = nil
	})

	return rw
}

func (rw *NetworkWindow) Show() {
	// Mark window as open
	globalNetworkWindow = rw

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
	rw.ConfigurePasswordEntry(passwordEntry)

	// Add keyboard shortcuts
	nameEntry.OnSubmitted = func(text string) {
		passwordEntry.FocusGained()
	}

	passwordEntry.OnSubmitted = func(text string) {
		// Trigger create button when Enter is pressed on password field
		if nameEntry.Text != "" && rw.ValidatePassword(text) {
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
			dialog.ShowError(errors.New("network name cannot be empty"), rw.BaseWindow.Window)
			return
		}

		// Validate password using the abstract function
		if !rw.ValidatePassword(password) {
			dialog.ShowError(errors.New("password must be exactly 4 digits"), rw.BaseWindow.Window)
			return
		}

		// Show loading indicator
		createButton.SetText("Creating...")
		createButton.Disable()

		// Create network in a goroutine
		go func() {
			// Send create network command to backend
			res, err := rw.CreateNetwork(name, password)

			// Update UI on the main thread
			go func() {
				createButton.SetText("Create Network")
				createButton.Enable()

				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to create network: %v", err), rw.BaseWindow.Window)
					return
				}

				// Get network ID created by the server
				networkID := res.NetworkID

				if networkID != "" {
					// Save network to database
					err := rw.SaveNetwork(networkID, name, password)
					if err != nil {
						log.Printf("Error saving network to database: %v", err)
					}

					// Show success dialog with network ID
					networkIDEntry := widget.NewEntry()
					networkIDEntry.Text = networkID
					networkIDEntry.Disable()

					content := container.NewVBox(
						widget.NewLabel("Network created successfully!"),
						widget.NewLabel(""),
						widget.NewLabel("Network ID (share this with friends):"),
						networkIDEntry,
						widget.NewButton("Copy to Clipboard", func() {
							rw.BaseWindow.Window.Clipboard().SetContent(networkID)
							dialog.ShowInformation("Copied", "Network ID copied to clipboard", rw.Window)
						}),
					)

					successDialog := dialog.NewCustom("Success", "Close", content, rw.BaseWindow.Window)
					successDialog.Show()

					// Close the create window after showing success
					rw.BaseWindow.Close()
				} else {
					// If networkID is empty, it's likely an error occurred but wasn't caught
					dialog.ShowError(errors.New("failed to create network: no network ID returned"), rw.BaseWindow.Window)
				}
			}()
		}()
	})

	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		rw.BaseWindow.Close()
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

	rw.BaseWindow.SetContent(content)
	rw.BaseWindow.Show()

	// Set focus on the name field when window opens
	rw.BaseWindow.Window.Canvas().Focus(nameEntry)
}
