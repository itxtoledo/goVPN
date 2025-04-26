package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ConnectDialog representa um diálogo para conexão a uma sala
type ConnectDialog struct {
	UI            *UIManager
	Dialog        dialog.Dialog
	PasswordEntry *widget.Entry
}

// NewConnectDialog cria uma nova instância do diálogo de conexão
func NewConnectDialog(ui *UIManager) *ConnectDialog {
	cd := &ConnectDialog{
		UI: ui,
	}
	return cd
}

// Show exibe o diálogo de conexão
func (cd *ConnectDialog) Show() {
	// Obter a sala selecionada
	room := cd.UI.SelectedRoom
	if room == nil {
		log.Println("No room selected")
		return
	}

	content := widget.NewLabel("Connect to this room?")

	// Criar o diálogo
	cd.Dialog = dialog.NewCustomConfirm(
		"Connect to Room",
		"Connect",
		"Cancel",
		content,
		func(confirmed bool) {
			if confirmed {
				// Tentar conectar à sala
				cd.connectToRoom(room.ID, room.Password)
			}
		},
		cd.UI.MainWindow,
	)

	cd.Dialog.Show()
}

// connectToRoom conecta o cliente à sala especificada
func (cd *ConnectDialog) connectToRoom(roomID, password string) {
	// Verificar se o cliente está conectado
	if cd.UI.VPN.NetworkManager == nil || cd.UI.VPN.NetworkManager.GetConnectionState() != ConnectionStateConnected {
		dialog.ShowError(fmt.Errorf("not connected to server"), cd.UI.MainWindow)
		return
	}

	// Use goroutine to prevent UI freezing
	go func() {
		// Tentar conectar à sala
		err := cd.UI.VPN.NetworkManager.JoinRoom(roomID, password)

		// Use fyne.Do to safely update UI from a goroutine
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to join room: %v", err), cd.UI.MainWindow)
			} else {
				// Atualizar a interface
				cd.UI.refreshUI()
				dialog.ShowInformation("Success", "Connected to room successfully", cd.UI.MainWindow)
			}
		})
	}()
}
