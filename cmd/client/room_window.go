package main

import (
	"fmt"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// RoomDialog representa uma janela de diálogo para criar ou entrar em uma sala
type RoomDialog struct {
	UI        *UIManager
	Dialog    dialog.Dialog
	NameEntry *widget.Entry
	DescEntry *widget.Entry
	PassEntry *widget.Entry
	IsCreate  bool
}

// NewRoomDialog cria um novo diálogo para sala
func NewRoomDialog(ui *UIManager, isCreate bool) *RoomDialog {
	rd := &RoomDialog{
		UI:       ui,
		IsCreate: isCreate,
	}

	// Criar campos de entrada
	rd.NameEntry = widget.NewEntry()
	rd.NameEntry.SetPlaceHolder("Room Name")

	rd.DescEntry = widget.NewEntry()
	rd.DescEntry.SetPlaceHolder("Room Description")
	rd.DescEntry.MultiLine = true

	rd.PassEntry = widget.NewPasswordEntry()
	rd.PassEntry.SetPlaceHolder("Password (optional)")

	// Criar o formulário
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Room Name", Widget: rd.NameEntry, HintText: "Enter a name for the room"},
			{Text: "Description", Widget: rd.DescEntry, HintText: "Describe the purpose of the room"},
			{Text: "Password", Widget: rd.PassEntry, HintText: "Optional password for the room"},
		},
	}

	// Título do diálogo e texto do botão de acordo com a operação
	title := "Create Room"
	buttonText := "Create"
	if !isCreate {
		title = "Join Room"
		buttonText = "Join"
	}

	// Criar o diálogo
	rd.Dialog = dialog.NewCustom(
		title,
		buttonText,
		form,
		rd.UI.MainWindow,
	)

	// Configurar o botão de confirmação
	confirm, ok := rd.Dialog.(*dialog.CustomDialog)
	if ok {
		confirm.SetOnClosed(func() {
			rd.handleConfirm()
		})
	}

	return rd
}

// Show exibe o diálogo
func (rd *RoomDialog) Show() {
	rd.Dialog.Show()
}

// handleConfirm trata a confirmação do diálogo
func (rd *RoomDialog) handleConfirm() {
	// Validar campos
	if rd.NameEntry.Text == "" {
		dialog.ShowError(fmt.Errorf("room name is required"), rd.UI.MainWindow)
		return
	}

	if rd.IsCreate {
		// Criar sala
		err := rd.UI.VPN.NetworkManager.CreateRoom(rd.NameEntry.Text, rd.DescEntry.Text, rd.PassEntry.Text)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to create room: %v", err), rd.UI.MainWindow)
			return
		}

		// Atualizar lista de salas
		rd.UI.refreshNetworkList()
	} else {
		// Entrar na sala
		err := rd.UI.VPN.NetworkManager.JoinRoom(rd.NameEntry.Text, rd.PassEntry.Text)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to join room: %v", err), rd.UI.MainWindow)
			return
		}

		// Atualizar UI
		rd.UI.refreshUI()
	}
}
