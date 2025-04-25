package main

import "github.com/itxtoledo/govpn/cmd/client/dialogs"

// ConnectDialog manages the connect to network interface as a dialog
type ConnectDialog struct {
	UI     *UIManager
	dialog *dialogs.ConnectDialog
}

// NewConnectDialog creates a new connect dialog
func NewConnectDialog(ui *UIManager) *ConnectDialog {
	connectDialog := &ConnectDialog{
		UI: ui,
	}

	// Create the underlying dialog from our dialogs package
	var dialogRef interface{} = connectDialog
	connectDialog.dialog = dialogs.NewConnectDialog(
		ui.MainWindow,
		&dialogRef,
		ui.ShowMessage,
	)

	return connectDialog
}

// Show displays the connect dialog
func (cd *ConnectDialog) Show() {
	// Define a connection function that will attempt to join the room
	// and handle any errors that occur during the connection process
	connectToRoom := func(roomID, password string) error {
		// Use the NetworkManager to join the room
		if cd.UI != nil && cd.UI.VPN != nil && cd.UI.VPN.NetworkManager != nil {
			return cd.UI.VPN.NetworkManager.JoinRoom(roomID, password)
		}
		return nil
	}

	cd.dialog.Show(ValidatePassword, ConfigurePasswordEntry, connectToRoom)
}
