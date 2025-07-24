package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/data"
)

// ConnectDialogManager é a interface que define as operações necessárias para o diálogo de conexão
type ConnectDialogManager interface {
	GetSelectedNetwork() *data.Network
	ConnectToNetwork(networkID, computername string) error
	GetMainWindow() fyne.Window
}

// ConnectDialog representa um diálogo para conexão a uma sala
type ConnectDialog struct {
	UI     ConnectDialogManager
	Dialog dialog.Dialog

	ComputerName string
}

// NewConnectDialog cria uma nova instância do diálogo de conexão
func NewConnectDialog(ui ConnectDialogManager, computername string) *ConnectDialog {
	cd := &ConnectDialog{
		UI:           ui,
		ComputerName: computername,
	}
	return cd
}

// Show exibe o diálogo de conexão
func (cd *ConnectDialog) Show() {
	// Obter a sala selecionada
	network := cd.UI.GetSelectedNetwork()
	if network == nil {
		return
	}

	networkID := network.NetworkID

	// Criar o diálogo
	cd.Dialog = dialog.NewCustomConfirm(
		"Connect to Network",
		"Connect",
		"Cancel",
		widget.NewLabel("Connect to this network?"),
		func(confirmed bool) {
			if confirmed {
				// Tentar conectar à sala
				err := cd.UI.ConnectToNetwork(networkID, cd.ComputerName)
				if err != nil {
					// Exibir erro em caso de falha
					errorDialog := dialog.NewError(err, cd.UI.GetMainWindow())
					errorDialog.Show()
				}
			}
		},
		cd.UI.GetMainWindow(),
	)

	cd.Dialog.Show()
}
