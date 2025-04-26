package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// CreateRoomDialog represents a dialog for creating a new room
type CreateRoomDialog struct {
	UI         *UIManager
	MainWindow fyne.Window
	FormDialog dialog.Dialog
	NameEntry  *widget.Entry
	PassEntry  *widget.Entry
}

// NewCreateRoomDialog creates a new instance of the room creation dialog
func NewCreateRoomDialog(ui *UIManager) *CreateRoomDialog {
	crd := &CreateRoomDialog{
		UI:         ui,
		MainWindow: ui.MainWindow,
	}

	// Create form inputs
	crd.NameEntry = widget.NewEntry()

	// Create password entry with validation for digits
	crd.PassEntry = widget.NewPasswordEntry()

	// Create form items
	items := []*widget.FormItem{
		widget.NewFormItem("Room Name", crd.NameEntry),
		widget.NewFormItem("Password", crd.PassEntry),
	}

	// Create the form dialog
	crd.FormDialog = dialog.NewForm("Create Room", "Create", "Cancel", items,
		func(submitted bool) {
			if !submitted {
				return
			}

			// Process form submission
			name := crd.NameEntry.Text
			pass := crd.PassEntry.Text

			// Validate input
			if name == "" {
				dialog.ShowError(fmt.Errorf("room name is required"), crd.MainWindow)
				return
			}

			// Validate password using the models package
			if !models.ValidatePassword(pass) {
				dialog.ShowError(fmt.Errorf("password must be exactly 4 digits"), crd.MainWindow)
				return
			}

			// Verify network connection
			if crd.UI.VPN.NetworkManager == nil || crd.UI.VPN.NetworkManager.GetConnectionState() != ConnectionStateConnected {
				dialog.ShowError(fmt.Errorf("not connected to server"), crd.MainWindow)
				return
			}

			// Show progress dialog
			progress := dialog.NewProgress("Creating Room", "Creating new room, please wait...", crd.MainWindow)
			progress.Show()

			// Create room in a goroutine
			go func() {
				// Create room
				log.Printf("Creating room: %s", name)
				err := crd.UI.VPN.NetworkManager.CreateRoom(name, pass)

				// Update UI using fyne.Do
				fyne.Do(func() {
					// Hide progress dialog
					progress.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to create room: %v", err), crd.MainWindow)
						return
					}

					// Update room list
					crd.UI.refreshNetworkList()

					// Show success message
					dialog.ShowInformation("Success", "Room created successfully", crd.MainWindow)
				})
			}()
		}, crd.MainWindow)

	return crd
}

// Show displays the create room dialog
func (crd *CreateRoomDialog) Show() {
	crd.FormDialog.Show()
}

// JoinRoomDialog represents a dialog for joining an existing room
type JoinRoomDialog struct {
	UI         *UIManager
	MainWindow fyne.Window
	FormDialog dialog.Dialog
	IDEntry    *widget.Entry
	PassEntry  *widget.Entry
}

// NewJoinRoomDialog creates a new instance of the room joining dialog
func NewJoinRoomDialog(ui *UIManager) *JoinRoomDialog {
	jrd := &JoinRoomDialog{
		UI:         ui,
		MainWindow: ui.MainWindow,
	}

	// Create form inputs
	jrd.IDEntry = widget.NewEntry()
	jrd.PassEntry = widget.NewPasswordEntry()

	// Create form items
	items := []*widget.FormItem{
		widget.NewFormItem("Room ID", jrd.IDEntry),
		widget.NewFormItem("Password", jrd.PassEntry),
	}

	// Create the form dialog
	jrd.FormDialog = dialog.NewForm("Join Room", "Join", "Cancel", items,
		func(submitted bool) {
			if !submitted {
				return
			}

			// Process form submission
			roomID := jrd.IDEntry.Text
			pass := jrd.PassEntry.Text

			// Validate input
			if roomID == "" {
				dialog.ShowError(fmt.Errorf("room id is required"), jrd.MainWindow)
				return
			}

			// Verify network connection
			if jrd.UI.VPN.NetworkManager == nil || jrd.UI.VPN.NetworkManager.GetConnectionState() != ConnectionStateConnected {
				dialog.ShowError(fmt.Errorf("not connected to server"), jrd.MainWindow)
				return
			}

			// Show progress dialog
			progress := dialog.NewProgress("Joining Room", "Joining room, please wait...", jrd.MainWindow)
			progress.Show()

			// Join room in a goroutine
			go func() {
				// Join room
				log.Printf("Joining room: %s", roomID)
				err := jrd.UI.VPN.NetworkManager.JoinRoom(roomID, pass)

				// Update UI using fyne.Do
				fyne.Do(func() {
					// Hide progress dialog
					progress.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to join room: %v", err), jrd.MainWindow)
						return
					}

					// Update interface
					jrd.UI.refreshUI()

					// Show success message
					dialog.ShowInformation("Success", "Joined room successfully", jrd.MainWindow)
				})
			}()
		}, jrd.MainWindow)

	return jrd
}

// Show displays the join room dialog
func (jrd *JoinRoomDialog) Show() {
	jrd.FormDialog.Show()
}
