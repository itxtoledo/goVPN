package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

	roomName := room.Name
	passwordRequired := room.Password != ""

	var content fyne.CanvasObject

	if passwordRequired {
		// Se a sala requer senha, mostrar campo de senha
		cd.PasswordEntry = widget.NewPasswordEntry()
		cd.PasswordEntry.PlaceHolder = "Enter room password"

		content = container.NewVBox(
			widget.NewLabel("This room requires a password:"),
			cd.PasswordEntry,
		)
	} else {
		// Se a sala não requer senha, mostrar apenas uma confirmação
		content = widget.NewLabel("Connect to this room?")
	}

	// Criar o diálogo
	cd.Dialog = dialog.NewCustomConfirm(
		"Connect to Room",
		"Connect",
		"Cancel",
		content,
		func(confirmed bool) {
			if confirmed {
				password := ""
				if cd.PasswordEntry != nil {
					password = cd.PasswordEntry.Text
				}

				// Tentar conectar à sala
				cd.connectToRoom(roomName, password)
			}
		},
		cd.UI.MainWindow,
	)

	cd.Dialog.Show()
}

// connectToRoom conecta o cliente à sala especificada
func (cd *ConnectDialog) connectToRoom(roomName, password string) {
	// Verificar se o cliente está conectado
	if cd.UI.VPN.NetworkManager == nil || cd.UI.VPN.NetworkManager.GetConnectionState() != ConnectionStateConnected {
		dialog.ShowError(fmt.Errorf("not connected to server"), cd.UI.MainWindow)
		return
	}

	// Tentar conectar à sala
	err := cd.UI.VPN.NetworkManager.JoinRoom(roomName, password)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to join room: %v", err), cd.UI.MainWindow)
	} else {
		// Atualizar a interface
		cd.UI.refreshUI()
	}
}
