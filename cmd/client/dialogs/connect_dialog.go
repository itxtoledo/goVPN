package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ConnectDialogManager é a interface que define as operações necessárias para o diálogo de conexão
type ConnectDialogManager interface {
	GetSelectedRoom() interface{}
	ConnectToRoom(roomName, password string) error
}

// ConnectDialog representa um diálogo para conexão a uma sala
type ConnectDialog struct {
	UI            ConnectDialogManager
	Dialog        dialog.Dialog
	PasswordEntry *widget.Entry
}

// NewConnectDialog cria uma nova instância do diálogo de conexão
func NewConnectDialog(ui ConnectDialogManager) *ConnectDialog {
	cd := &ConnectDialog{
		UI: ui,
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

	// O sistema de tipos Go não nos permite acessar diretamente os campos da sala
	// através da interface, então assumimos que o tipo de retorno é um mapa
	roomData, ok := room.(map[string]interface{})
	if !ok {
		return
	}

	roomName, _ := roomData["Name"].(string)
	passwordRequired := roomData["Password"] != nil && roomData["Password"].(string) != ""

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
				err := cd.UI.ConnectToRoom(roomName, password)
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
