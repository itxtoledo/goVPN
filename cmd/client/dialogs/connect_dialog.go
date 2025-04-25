package dialogs

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ConnectDialog manages the connect to network interface as a dialog
type ConnectDialog struct {
	MainWindow       fyne.Window
	ConnectDialogRef *interface{}
	ShowMessage      func(title, message string)
}

// NewConnectDialog creates a new connect dialog
func NewConnectDialog(
	mainWindow fyne.Window,
	connectDialogRef *interface{},
	showMessage func(title, message string),
) *ConnectDialog {
	return &ConnectDialog{
		MainWindow:       mainWindow,
		ConnectDialogRef: connectDialogRef,
		ShowMessage:      showMessage,
	}
}

// Show displays the connect dialog using the form dialog approach
func (cd *ConnectDialog) Show(
	validatePassword func(string) bool,
	configurePasswordEntry func(*widget.Entry),
	connectToRoom func(roomID, password string) error,
) {
	// Create form inputs
	networkIDEntry := widget.NewEntry()
	passwordEntry := widget.NewPasswordEntry()

	// Apply reusable password validation configuration
	configurePasswordEntry(passwordEntry)

	// Create form items
	items := []*widget.FormItem{
		widget.NewFormItem("ID", networkIDEntry),
		widget.NewFormItem("Pass", passwordEntry),
	}

	// Show the form dialog
	formDialog := dialog.NewForm("Connect to Network", "Connect", "Cancel", items, func(submitted bool) {
		if !submitted {
			// Dialog was cancelled
			*cd.ConnectDialogRef = nil
			return
		}

		// Process form submission
		networkID := networkIDEntry.Text
		password := passwordEntry.Text

		// Basic validation
		if networkID == "" {
			dialog.ShowError(errors.New("Network ID cannot be empty"), cd.MainWindow)
			return
		}

		// Validate password using the abstract function
		if !validatePassword(password) {
			dialog.ShowError(errors.New("Password must be exactly 4 digits"), cd.MainWindow)
			return
		}

		// Show connection in progress message
		cd.ShowMessage("Connection", "Attempting to connect to room: "+networkID)

		// Attempt to connect using the provided connection function
		if err := connectToRoom(networkID, password); err != nil {
			// If connection fails, show the error
			dialog.ShowError(errors.New("Connection failed: "+err.Error()), cd.MainWindow)
			return
		}

		// Clear the reference only on successful connection
		*cd.ConnectDialogRef = nil
	}, cd.MainWindow)

	// Define a minimum width for the dialog
	formDialog.Resize(fyne.NewSize(300, 200))
	formDialog.Show()
}
