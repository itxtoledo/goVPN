package main

import (
	"fmt"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ConnectWindow gerencia a interface de conexão à VPN
type ConnectWindow struct {
	UI            *UIManager
	Window        fyne.Window
	RoomIDEntry   *widget.Entry
	PasswordEntry *widget.Entry
	ConnectButton *widget.Button
	Container     *fyne.Container
	Status        *widget.Label
}

// NewConnectWindow cria uma nova janela de conexão
func NewConnectWindow(ui *UIManager) *ConnectWindow {
	connectWindow := &ConnectWindow{
		UI:     ui,
		Window: ui.createWindow("Conectar à VPN", 400, 300, false),
	}

	// Adiciona manipulador para quando a janela for fechada
	connectWindow.Window.SetOnClosed(func() {
		connectWindow.Window = nil
	})

	return connectWindow
}

// Show exibe a janela de conexão
func (cw *ConnectWindow) Show() {
	// Se a janela já foi fechada, recria
	if cw.Window == nil {
		cw.Window = cw.UI.createWindow("Conectar à VPN", 400, 300, false)
		// Re-adiciona o manipulador para quando a janela for fechada
		cw.Window.SetOnClosed(func() {
			cw.Window = nil
		})
	}

	// Garante que o conteúdo é criado antes de exibir a janela
	content := cw.CreateContent()

	// Define o conteúdo da janela
	cw.Window.SetContent(content)

	// Limpa os campos
	cw.RoomIDEntry.SetText("")
	cw.PasswordEntry.SetText("")
	cw.Status.SetText("")

	// Exibe a janela centralizada
	cw.Window.CenterOnScreen()
	cw.Window.Show()
}

// CreateContent cria o conteúdo da janela de conexão
func (cw *ConnectWindow) CreateContent() fyne.CanvasObject {
	// Campos de entrada
	cw.RoomIDEntry = widget.NewEntry()
	cw.RoomIDEntry.SetPlaceHolder("ID da Sala")

	cw.PasswordEntry = widget.NewPasswordEntry()
	cw.PasswordEntry.SetPlaceHolder("Senha da Sala")

	// Restringir para apenas números e máximo de 4 caracteres
	cw.PasswordEntry.Validator = func(s string) error {
		if len(s) > 4 {
			return fmt.Errorf("a senha deve ter no máximo 4 caracteres")
		}
		for _, r := range s {
			if !unicode.IsDigit(r) {
				return fmt.Errorf("a senha deve conter apenas números")
			}
		}
		return nil
	}

	// Limitar entrada para 4 caracteres em tempo real
	cw.PasswordEntry.OnChanged = func(s string) {
		if len(s) > 4 {
			cw.PasswordEntry.SetText(s[:4])
		}
	}

	// Status de conexão
	cw.Status = widget.NewLabel("")

	// Botões de ação
	cw.ConnectButton = widget.NewButton("Conectar", func() {
		cw.connect()
	})

	// Formulário de conexão
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "ID da Sala", Widget: cw.RoomIDEntry},
			{Text: "Senha", Widget: cw.PasswordEntry},
		},
		SubmitText: "Conectar",
		OnSubmit: func() {
			cw.connect()
		},
	}

	// Container principal
	cw.Container = container.NewVBox(
		widget.NewLabel("Conectar a uma Sala VPN"),
		form,
		cw.Status,
		container.NewHBox(
			widget.NewButton("Cancelar", func() {
				cw.Window.Close()
			}),
		),
	)

	return cw.Container
}

// connect tenta conectar a uma sala VPN
func (cw *ConnectWindow) connect() {
	// Validação de campos
	if cw.RoomIDEntry.Text == "" || cw.PasswordEntry.Text == "" {
		cw.Status.SetText("Preencha todos os campos")
		return
	}

	// Tenta conectar ao servidor de sinalização se ainda não estiver conectado
	if !cw.UI.VPN.NetworkManager.IsConnected {
		cw.Status.SetText("Conectando ao servidor...")
		err := cw.UI.VPN.NetworkManager.Connect()
		if err != nil {
			cw.Status.SetText("Falha na conexão: " + err.Error())
			return
		}
	}

	// Tenta entrar na sala
	cw.Status.SetText("Entrando na sala...")
	err := cw.UI.VPN.NetworkManager.JoinRoom(cw.RoomIDEntry.Text, cw.PasswordEntry.Text)
	if err != nil {
		cw.Status.SetText("Falha ao entrar na sala: " + err.Error())
		return
	}

	// Guarda as informações da sala e fecha a janela
	cw.UI.VPN.CurrentRoom = cw.RoomIDEntry.Text
	cw.UI.VPN.NetworkManager.RoomName = cw.RoomIDEntry.Text
	cw.UI.VPN.IsConnected = true

	// Atualiza o estado do botão toggle no header e as informações de conexão
	cw.UI.updatePowerButtonState()
	cw.UI.updateIPInfo()
	cw.UI.updateRoomName()

	cw.Window.Close()
}
