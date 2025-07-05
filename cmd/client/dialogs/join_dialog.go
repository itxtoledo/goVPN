package dialogs

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// JoinDialog manages the room joining interface as a dialog
type JoinDialog struct {
	MainWindow fyne.Window
	JoinRoom   func(string, string, string) (*models.JoinRoomResponse, error)
	SaveRoom   func(string, string, string) error
	Username   string
}

// NewJoinDialog creates a new room joining dialog
func NewJoinDialog(
	mainWindow fyne.Window,
	joinRoom func(string, string, string) (*models.JoinRoomResponse, error),
	saveRoom func(string, string, string) error,
	username string,
	validatePassword func(string) bool,
	configurePasswordEntry func(*widget.Entry),
) *JoinDialog {
	return &JoinDialog{
		MainWindow: mainWindow,
		JoinRoom:   joinRoom,
		SaveRoom:   saveRoom,
		Username:   username,
	}
}

// Show displays the room joining dialog using the form dialog approach
func (jd *JoinDialog) Show(validatePassword func(string) bool, configurePasswordEntry func(*widget.Entry)) {
	// Create form inputs
	roomIDEntry := widget.NewEntry()
	roomIDEntry.PlaceHolder = "Enter Room ID"

	passwordEntry := widget.NewPasswordEntry()
	configurePasswordEntry(passwordEntry)

	items := []*widget.FormItem{
		widget.NewFormItem("Room ID", roomIDEntry),
		widget.NewFormItem("Password", passwordEntry),
	}

	formDialog := dialog.NewForm("Join Network", "Join", "Cancel", items, func(submitted bool) {
		if !submitted {
			return
		}

		roomID := roomIDEntry.Text
		password := passwordEntry.Text

		if roomID == "" {
			dialog.ShowError(errors.New("Room ID cannot be empty"), jd.MainWindow)
			return
		}

		if !validatePassword(password) {
			dialog.ShowError(errors.New("Password must be exactly 4 digits"), jd.MainWindow)
			return
		}

		progressDialog := dialog.NewCustom("Joining Network", "Cancel", widget.NewLabel("Joining network, please wait..."), jd.MainWindow)
		progressDialog.Show()

		go func() {
			_, err := jd.JoinRoom(roomID, password, jd.Username)

			fyne.Do(func() {
				progressDialog.Dismiss()

				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to join network: %v", err), jd.MainWindow)
					return
				}

				// Optionally save the room if needed, though JoinRoom might handle this
				// err = jd.SaveRoom(roomID, "Joined Room", password) // "Joined Room" is a placeholder name
				// if err != nil {
				// 	log.Printf("Error saving joined room to database: %v", err)
				// }

				dialog.ShowInformation("Success", "Successfully joined network!", jd.MainWindow)
			})
		}()
	}, jd.MainWindow)

	formDialog.Resize(jd.MainWindow.Canvas().Size())
	formDialog.Show()
}
