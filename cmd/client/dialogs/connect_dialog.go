package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// ConnectDialogManager é a interface que define as operações necessárias para o diálogo de conexão
type ConnectDialogManager interface {
	GetSelectedRoom() *storage.Room
	ConnectToRoom(roomID, username string) error
}

// ConnectDialog representa um diálogo para conexão a uma sala
type ConnectDialog struct {
	UI            ConnectDialogManager
	Dialog        dialog.Dialog
	
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

	// Criar o diálogo
	cd.Dialog = dialog.NewCustomConfirm(
		"Connect to Room",
		"Connect",
		"Cancel",
		widget.NewLabel("Connect to this room?"),
		func(confirmed bool) {
			if confirmed {
				// Tentar conectar à sala
				err := cd.UI.ConnectToRoom(roomID, cd.Username)
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
