package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// LoginWindow gerencia a interface de login
type LoginWindow struct {
	UI            *UIManager
	Window        fyne.Window
	ServerEntry   *widget.Entry
	UsernameEntry *widget.Entry
	PasswordEntry *widget.Entry
	LoginButton   *widget.Button
	Container     *fyne.Container
}

// NewLoginWindow cria uma nova janela de login
func NewLoginWindow(ui *UIManager) *LoginWindow {
	loginWindow := &LoginWindow{
		UI:     ui,
		Window: ui.createWindow("Login - goVPN", 400, 300, false),
	}

	return loginWindow
}

// Show exibe a janela de login
func (lw *LoginWindow) Show() {
	// Se a janela já foi fechada, recria
	if lw.Window == nil {
		lw.Window = lw.UI.createWindow("Login - goVPN", 400, 300, false)
	}

	// Inicializa os componentes necessários antes de exibir a janela
	content := lw.CreateContent()

	// Define o conteúdo da janela
	lw.Window.SetContent(content)

	// Exibe a janela centralizada
	lw.Window.CenterOnScreen()
	lw.Window.Show()
}

// CreateContent cria o conteúdo da janela de login
func (lw *LoginWindow) CreateContent() fyne.CanvasObject {
	// Campos de entrada
	lw.ServerEntry = widget.NewEntry()
	lw.ServerEntry.SetPlaceHolder("Endereço do servidor de sinalização")
	lw.ServerEntry.SetText(lw.UI.VPN.NetworkManager.SignalServer)

	lw.UsernameEntry = widget.NewEntry()
	lw.UsernameEntry.SetPlaceHolder("Nome de usuário")

	lw.PasswordEntry = widget.NewPasswordEntry()
	lw.PasswordEntry.SetPlaceHolder("Senha")

	// Botão de login
	lw.LoginButton = widget.NewButton("Conectar", func() {
		lw.login()
	})

	// Formulário de login
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Servidor", Widget: lw.ServerEntry},
			{Text: "Usuário", Widget: lw.UsernameEntry},
			{Text: "Senha", Widget: lw.PasswordEntry},
		},
		SubmitText: "Conectar",
		OnSubmit:   lw.login,
	}

	// Container principal
	lw.Container = container.NewVBox(
		widget.NewLabel("Login no Servidor goVPN"),
		form,
		container.NewHBox(
			widget.NewButton("Cancelar", func() {
				lw.Window.Hide()
			}),
		),
	)

	return lw.Container
}

// login processa a tentativa de login
func (lw *LoginWindow) login() {
	// Atualiza o endereço do servidor de sinalização
	lw.UI.VPN.NetworkManager.SignalServer = lw.ServerEntry.Text

	// Aqui seria implementada a lógica de autenticação real
	// Para este exemplo, apenas tentamos conectar ao servidor
	err := lw.UI.VPN.NetworkManager.Connect()
	if err != nil {
		lw.UI.ShowMessage("Erro", "Não foi possível conectar ao servidor: "+err.Error())
		return
	}

	// Esconde a janela de login após sucesso
	lw.Window.Hide()
}
