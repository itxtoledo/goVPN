package dialogs

import (
	"errors"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// RoomDialog manages the room (network) creation interface as a dialog
type RoomDialog struct {
	UI            interface{} // Will be UIManager, but we use interface{} to avoid import cycle
	MainWindow    fyne.Window
	CreateRoom    func(string, string) error
	SaveRoom      func(string, string, string) error
	GetRoomID     func() string
	RoomDialogRef *interface{} // Will be used to clean up reference in UIManager
}

// NewRoomDialog creates a new room creation dialog
func NewRoomDialog(
	ui interface{},
	mainWindow fyne.Window,
	createRoom func(string, string) error,
	saveRoom func(string, string, string) error,
	getRoomID func() string,
	roomDialogRef *interface{},
	validatePassword func(string) bool,
	configurePasswordEntry func(*widget.Entry),
) *RoomDialog {

	return &RoomDialog{
		UI:            ui,
		MainWindow:    mainWindow,
		CreateRoom:    createRoom,
		SaveRoom:      saveRoom,
		GetRoomID:     getRoomID,
		RoomDialogRef: roomDialogRef,
	}
}

// Show displays the room creation dialog using the form dialog approach
func (rd *RoomDialog) Show(validatePassword func(string) bool, configurePasswordEntry func(*widget.Entry)) {
	// Create form inputs
	nameEntry := widget.NewEntry()

	// Password entry with validation for 4 digits
	passwordEntry := widget.NewPasswordEntry()

	// Apply reusable password validation configuration
	configurePasswordEntry(passwordEntry)

	// Create form items with shorter labels
	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Pass", passwordEntry),
	}

	// Show the form dialog
	formDialog := dialog.NewForm("Create Network", "Create", "Cancel", items, func(submitted bool) {
		if !submitted {
			// Dialog was cancelled
			*rd.RoomDialogRef = nil
			return
		}

		// Process form submission
		name := nameEntry.Text
		password := passwordEntry.Text

		// Validate name
		if name == "" {
			dialog.ShowError(errors.New("Network name cannot be empty"), rd.MainWindow)
			return
		}

		// Validate password using the abstract function
		if !validatePassword(password) {
			dialog.ShowError(errors.New("Password must be exactly 4 digits"), rd.MainWindow)
			return
		}

		// Show progress dialog
		progressDialog := dialog.NewCustom("Creating Network", "Cancel", widget.NewLabel("Creating network, please wait..."), rd.MainWindow)
		progressDialog.Show()

		// Create network in a goroutine
		go func() {
			// Send create room command to backend
			err := rd.CreateRoom(name, password)

			// Update UI on the main thread
			fyne.Do(func() {
				// Hide progress dialog
				progressDialog.Dismiss()

				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to create network: %v", err), rd.MainWindow)
					// Clear the reference and return early to avoid showing success dialog
					*rd.RoomDialogRef = nil
					return
				}

				// Get room ID created by the server
				roomID := rd.GetRoomID()

				if roomID != "" {
					// Save room to database
					err := rd.SaveRoom(roomID, name, password)
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
							rd.MainWindow.Clipboard().SetContent(roomID)
							dialog.ShowInformation("Copied", "Network ID copied to clipboard", rd.MainWindow)
						}),
					)

					successDialog := dialog.NewCustom("Success", "Close", content, rd.MainWindow)
					successDialog.Show()
				} else {
					// If roomID is empty, it's likely an error occurred but wasn't caught
					dialog.ShowError(errors.New("Failed to create network: No network ID returned"), rd.MainWindow)
				}

				// Clear the reference
				// *rd.RoomDialogRef = nil
			})
		}()
	}, rd.MainWindow)

	// Define a minimum width for the dialog
	formDialog.Resize(fyne.NewSize(300, 200))
	formDialog.Show()
}
