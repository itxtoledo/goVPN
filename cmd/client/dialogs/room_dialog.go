package dialogs

import (
	"errors"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// RoomDialog manages the room (network) creation interface as a dialog
type RoomDialog struct {
	MainWindow fyne.Window
	CreateRoom func(string, string) (*models.CreateRoomResponse, error)
	SaveRoom   func(string, string, string) error
	GetRoomID  func() string
	Username   string
}

// NewRoomDialog creates a new room creation dialog
func NewRoomDialog(
	mainWindow fyne.Window,
	createRoom func(string, string) (*models.CreateRoomResponse, error),
	saveRoom func(string, string, string) error,
	getRoomID func() string,
	username string,
	validatePassword func(string) bool,
	configurePasswordEntry func(*widget.Entry),
) *RoomDialog {

	return &RoomDialog{
		MainWindow: mainWindow,
		CreateRoom: createRoom,
		SaveRoom:   saveRoom,
		GetRoomID:  getRoomID,
		Username:   username,
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
			res, err := rd.CreateRoom(name, password)

			// Update UI on the main thread
			fyne.Do(func() {
				// Hide progress dialog
				progressDialog.Dismiss()

				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to create network: %v", err), rd.MainWindow)
					return
				}

				// Get room ID created by the server
				roomID := res.RoomID

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
			})
		}()
	}, rd.MainWindow)

	// Resize the dialog to occupy the full width of the parent window
	formDialog.Resize(rd.MainWindow.Canvas().Size())
	formDialog.Show()
}
