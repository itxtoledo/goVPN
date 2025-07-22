package main

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	smodels "github.com/itxtoledo/govpn/libs/signaling/models"
)

// Global variable to ensure only one network window can be open
var globalNetworkWindow *NetworkWindow

// NetworkWindow manages the network (network) creation interface as a window
type NetworkWindow struct {
	*BaseWindow
	CreateNetwork          func(string, string) (*smodels.CreateNetworkResponse, error)
	GetNetworkID           func() string
	ComputerName           string
	ValidatePIN       func(string) bool
	ConfigurePINEntry func(*widget.Entry)
	OnNetworkCreated       func(networkID, networkName, pin string)
}

// NewNetworkWindow creates a new network creation window
func NewNetworkWindow(
	app fyne.App,
	createNetwork func(string, string) (*smodels.CreateNetworkResponse, error),
	getNetworkID func() string,
	computername string,
	validatePIN func(string) bool,
	configurePINEntry func(*widget.Entry),
	onNetworkCreated func(networkID, networkName, pin string),
) *NetworkWindow {
	rw := &NetworkWindow{
		BaseWindow:             NewBaseWindow(app, "Create Network", 320, 260),
		CreateNetwork:          createNetwork,
		GetNetworkID:           getNetworkID,
		ComputerName:           computername,
		ValidatePIN:       validatePIN,
		ConfigurePINEntry: configurePINEntry,
		OnNetworkCreated:       onNetworkCreated,
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

	pinEntry := widget.NewPasswordEntry()
	pinEntry.PlaceHolder = "4-digit PIN"
	rw.ConfigurePINEntry(pinEntry)

	confirmPINEntry := widget.NewPasswordEntry()
	confirmPINEntry.PlaceHolder = "Repeat 4-digit PIN"
	rw.ConfigurePINEntry(confirmPINEntry)

	// Add keyboard shortcuts
	nameEntry.OnSubmitted = func(text string) {
		pinEntry.FocusGained()
	}

	pinEntry.OnSubmitted = func(text string) {
		confirmPINEntry.FocusGained()
	}

	confirmPINEntry.OnSubmitted = func(text string) {
		// Trigger create button when Enter is pressed on pin field
		if nameEntry.Text != "" && rw.ValidatePIN(text) {
			// Will be triggered by the create button logic
		}
	}

	// Create compact form with better spacing
	formContainer := container.NewVBox(
		widget.NewLabel("Network Name:"),
		container.NewPadded(nameEntry),
		widget.NewLabel("PIN:"),
		container.NewPadded(pinEntry),
		widget.NewLabel("Repeat PIN:"),
		container.NewPadded(confirmPINEntry),
	)

	// Create buttons
	var createButton *widget.Button
	createButton = widget.NewButtonWithIcon("Create Network", theme.ConfirmIcon(), func() {
		name := nameEntry.Text
		pin := pinEntry.Text
		confirmPIN := confirmPINEntry.Text

		// Validate name
		if name == "" {
			dialog.ShowError(errors.New("network name cannot be empty"), rw.BaseWindow.Window)
			return
		}

		// Validate pin using the abstract function
		if !rw.ValidatePIN(pin) {
			dialog.ShowError(errors.New("PIN must be exactly 4 digits"), rw.BaseWindow.Window)
			return
		}

		// Validate pin confirmation
		if pin != confirmPIN {
			dialog.ShowError(errors.New("PINs do not match"), rw.BaseWindow.Window)
			return
		}

		// Show loading indicator
		createButton.SetText("Creating...")
		createButton.Disable()

		// Create network in a goroutine
		go func() {
			// Send create network command to backend
			res, err := rw.CreateNetwork(name, pin)

			fyne.Do(func() {
				createButton.SetText("Create Network")
				createButton.Enable()

				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to create network: %v", err), rw.BaseWindow.Window)
					return
				}

				// Get network ID created by the server
				networkID := res.NetworkID

				if networkID != "" {
					// Invoke the callback with the network details
					rw.OnNetworkCreated(networkID, name, pin)

					// Close the create window after invoking the callback
					rw.BaseWindow.Close()
				} else {
					// If networkID is empty, it's likely an error occurred but wasn't caught
					dialog.ShowError(errors.New("failed to create network: no network ID returned"), rw.BaseWindow.Window)
				}
			})
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
