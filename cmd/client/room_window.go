package main

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
	UI *UIManager
}

// NewRoomDialog creates a new room creation dialog
func NewRoomDialog(ui *UIManager) *RoomDialog {
	roomDialog := &RoomDialog{
		UI: ui,
	}

	return roomDialog
}

// Show displays the room creation dialog using the form dialog approach
func (rd *RoomDialog) Show() {
	// Create form inputs
	nameEntry := widget.NewEntry()

	// Password entry with validation for 4 digits
	passwordEntry := widget.NewPasswordEntry()

	// Aplicando a configuração de validação de senha reutilizável
	ConfigurePasswordEntry(passwordEntry)

	// Create form items with rótulos mais curtos
	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Pass", passwordEntry),
	}

	// Show the form dialog
	formDialog := dialog.NewForm("Create Network", "Create", "Cancel", items, func(submitted bool) {
		if !submitted {
			// Dialog was cancelled
			rd.UI.RoomDialog = nil
			return
		}

		// Process form submission
		name := nameEntry.Text
		password := passwordEntry.Text

		// Validate name
		if name == "" {
			dialog.ShowError(errors.New("Network name cannot be empty"), rd.UI.MainWindow)
			return
		}

		// Validate password usando a função abstrata
		if !ValidatePassword(password) {
			dialog.ShowError(errors.New("Password must be exactly 4 digits"), rd.UI.MainWindow)
			return
		}

		// Show progress dialog
		progressDialog := dialog.NewCustom("Creating Network", "Cancel", widget.NewLabel("Creating network, please wait..."), rd.UI.MainWindow)
		progressDialog.Show()

		// Create network in a goroutine
		go func() {
			// Send create room command to backend
			err := rd.UI.VPN.NetworkManager.CreateRoom(name, password)

			// Hide progress dialog
			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to create network: %v", err), rd.UI.MainWindow)
				return
			}

			// Get room ID created by the server
			roomID := rd.UI.VPN.CurrentRoom

			if roomID != "" {
				// Save room to database
				_, err := rd.UI.VPN.DB.Exec(
					"INSERT OR REPLACE INTO rooms (id, name, password, last_connected) VALUES (?, ?, ?, CURRENT_TIMESTAMP)",
					roomID, name, password,
				)
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
						rd.UI.MainWindow.Clipboard().SetContent(roomID)
						dialog.ShowInformation("Copied", "Network ID copied to clipboard", rd.UI.MainWindow)
					}),
				)

				successDialog := dialog.NewCustom("Success", "Close", content, rd.UI.MainWindow)
				successDialog.Show()

				// Update UI
				rd.UI.refreshNetworkList()
			} else {
				dialog.ShowInformation("Network Created", "Network created successfully!", rd.UI.MainWindow)
			}

			// Clear the reference
			rd.UI.RoomDialog = nil
		}()
	}, rd.UI.MainWindow)

	// Define uma largura mínima para o diálogo
	formDialog.Resize(fyne.NewSize(300, 200))
	formDialog.Show()
}
