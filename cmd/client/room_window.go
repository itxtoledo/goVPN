package main

import (
	"github.com/itxtoledo/govpn/cmd/client/dialogs"
)

// RoomDialog manages the room (network) creation interface as a dialog
type RoomDialog struct {
	UI     *UIManager
	dialog *dialogs.RoomDialog
}

// NewRoomDialog creates a new room creation dialog
func NewRoomDialog(ui *UIManager) *RoomDialog {
	roomDialog := &RoomDialog{
		UI: ui,
	}

	// Create the underlying dialog from our dialogs package
	var dialogRef interface{} = roomDialog
	roomDialog.dialog = dialogs.NewRoomDialog(
		ui,
		ui.MainWindow,
		ui.VPN.NetworkManager.CreateRoom,
		ui.VPN.DBManager.SaveRoom,
		func() string { return ui.VPN.CurrentRoom },
		&dialogRef,
		ValidatePassword,
		ConfigurePasswordEntry,
	)

	return roomDialog
}

// Show displays the room creation dialog
func (rd *RoomDialog) Show() {
	rd.dialog.Show(ValidatePassword, ConfigurePasswordEntry)
}
