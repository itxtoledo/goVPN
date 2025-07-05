package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// ConnectDialogManager é a interface que define as operações necessárias para o diálogo de conexão
type ConnectDialogManager interface {
	GetSelectedRoom() *storage.Room
	ConnectToRoom(roomID, password, username string) error
}

// ConnectDialog representa um diálogo para conexão a uma sala
type ConnectDialog struct {
	UI            ConnectDialogManager
	Dialog        dialog.Dialog
	PasswordEntry *widget.Entry
	Username      string
}

// NewConnectDialog cria uma nova instância do diálogo de conexão
func NewConnectDialog(ui ConnectDialogManager, username string) *ConnectDialog {
	cd := &ConnectDialog{
		UI:       ui,
		Username: username,
	}
	return cd
}

// Show exibe o diálogo de conexão
func (cd *ConnectDialog) Show() {
	// Obter a sala selecionada
	room := cd.UI.GetSelectedRoom()
	if room == nil {
		return
	}

	roomID := room.ID
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
				err := cd.UI.ConnectToRoom(roomID, password, cd.Username)
				if err != nil {
					// Exibir erro em caso de falha
					errorDialog := dialog.NewError(err, fyne.CurrentApp().Driver().AllWindows()[0])
					errorDialog.Show()
				}
			}
		},
		fyne.CurrentApp().Driver().AllWindows()[0],
	)

	cd.Dialog.Show()
}
