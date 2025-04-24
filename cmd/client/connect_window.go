package main

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ConnectDialog manages the connect to network interface as a dialog
type ConnectDialog struct {
	UI *UIManager
}

// NewConnectDialog creates a new connect dialog
func NewConnectDialog(ui *UIManager) *ConnectDialog {
	connectDialog := &ConnectDialog{
		UI: ui,
	}

	return connectDialog
}

// Show displays the connect dialog using the form dialog approach
func (cd *ConnectDialog) Show() {
	// Create form inputs
	networkIDEntry := widget.NewEntry()

	passwordEntry := widget.NewPasswordEntry()

	// Aplicando a configuração de validação de senha reutilizável
	ConfigurePasswordEntry(passwordEntry)

	// Create form items
	items := []*widget.FormItem{
		widget.NewFormItem("ID", networkIDEntry),
		widget.NewFormItem("Pass", passwordEntry),
	}

	// Show the form dialog
	formDialog := dialog.NewForm("Connect to Network", "Connect", "Cancel", items, func(submitted bool) {
		if !submitted {
			// Dialog was cancelled
			cd.UI.ConnectDialog = nil
			return
		}

		// Process form submission
		networkID := networkIDEntry.Text
		password := passwordEntry.Text

		// Validação básica
		if networkID == "" {
			dialog.ShowError(errors.New("Network ID cannot be empty"), cd.UI.MainWindow)
			return
		}

		// Validar senha usando a função abstrata
		if !ValidatePassword(password) {
			dialog.ShowError(errors.New("Password must be exactly 4 digits"), cd.UI.MainWindow)
			return
		}

		// A senha seria utilizada aqui para conectar à rede,
		// mas não estamos implementando isso agora.
		// Na implementação completa:
		// err := cd.UI.VPN.NetworkManager.JoinNetwork(networkID, password)

		// For now, we'll just show a message
		cd.UI.ShowMessage("Connection", "Attempting to connect to network: "+networkID)

		// Clear the reference
		cd.UI.ConnectDialog = nil
	}, cd.UI.MainWindow)

	// Define uma largura mínima para o diálogo
	formDialog.Resize(fyne.NewSize(300, 200))
	formDialog.Show()
}
