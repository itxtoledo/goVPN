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

	// Check if user is already a member of the room
	isUserInRoom := false
	for _, r := range cd.UI.Rooms {
		if r.ID == room.ID {
			isUserInRoom = true
			break
		}
	}

	var title, buttonLabel, message string

	// Check if user is already connected to this room
	isConnectedToRoom := cd.UI.VPN.NetworkManager.RoomID == room.ID

	if isConnectedToRoom {
		title = "Disconnect from Room"
		buttonLabel = "Disconnect"
		message = "Disconnect from this room? You'll remain a member but will be disconnected."
	} else if isUserInRoom {
		title = "Connect to Room"
		buttonLabel = "Connect"
		message = "Connect to this room? No password needed since you're already a member."
	} else {
		title = "Connect to Room"
		buttonLabel = "Connect"
		message = "Connect to this room? You'll need to enter the room password."
	}

	content := widget.NewLabel(message)

	// Criar o diálogo
	cd.Dialog = dialog.NewCustomConfirm(
		title,
		buttonLabel,
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

// connectToRoom conecta ou desconecta o cliente da sala especificada
func (cd *ConnectDialog) connectToRoom(roomID, password string) {
	// Verificar se o cliente está conectado
	if cd.UI.VPN.NetworkManager == nil || cd.UI.VPN.NetworkManager.GetConnectionState() != ConnectionStateConnected {
		dialog.ShowError(fmt.Errorf("not connected to server"), cd.UI.MainWindow)
		return
	}

	// Check if user is already connected to this room
	isConnectedToRoom := cd.UI.VPN.NetworkManager.RoomID == roomID

	if isConnectedToRoom {
		// If already connected to this room, disconnect
		log.Printf("User is disconnecting from room %s", roomID)

		// Use goroutine to prevent UI freezing
		go func() {
			err := cd.UI.VPN.NetworkManager.DisconnectRoom(roomID)

			// Use fyne.Do to safely update UI from a goroutine
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to disconnect from room: %v", err), cd.UI.MainWindow)
				} else {
					// Atualizar a interface
					cd.UI.refreshUI()
					dialog.ShowInformation("Success", "Disconnected from room successfully", cd.UI.MainWindow)
				}
			})
		}()
		return
	}

	// Verificar se o usuário já está associado a esta sala
	// Se sim, usa ConnectRoom em vez de JoinRoom (evita pedir senha novamente)
	isUserInRoom := false
	for _, room := range cd.UI.Rooms {
		if room.ID == roomID {
			isUserInRoom = true
			break
		}
	}

	// Use goroutine to prevent UI freezing
	go func() {
		var err error

		if isUserInRoom {
			// Se já está associado à sala, conecta sem senha
			log.Printf("User is already in room %s, connecting without password", roomID)
			err = cd.UI.VPN.NetworkManager.ConnectRoom(roomID)
		} else {
			// Se não está associado ainda, junta-se à sala com senha
			log.Printf("User is not in room %s yet, joining with password", roomID)
			err = cd.UI.VPN.NetworkManager.JoinRoom(roomID, password)
		}

		// Use fyne.Do to safely update UI from a goroutine
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to connect to room: %v", err), cd.UI.MainWindow)
			} else {
				// Atualizar a interface
				cd.UI.refreshUI()
				dialog.ShowInformation("Success", "Connected to room successfully", cd.UI.MainWindow)
			}
		})
	}()
}
