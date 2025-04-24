package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ConnectWindow manages the connect to network interface
type ConnectWindow struct {
	*BaseWindow
	NetworkIDEntry *widget.Entry
	PasswordEntry  *widget.Entry
	Container      *fyne.Container
}

// NewConnectWindow creates a new connect window
func NewConnectWindow(ui *UIManager) *ConnectWindow {
	connectWindow := &ConnectWindow{
		BaseWindow: NewBaseWindow(ui, "Connect to Network - goVPN", 400, 250, false),
	}

	return connectWindow
}

// CreateContent creates the content for the connect window
func (cw *ConnectWindow) CreateContent() fyne.CanvasObject {
	// Input fields
	cw.NetworkIDEntry = widget.NewEntry()
	cw.NetworkIDEntry.SetPlaceHolder("Network ID")

	cw.PasswordEntry = widget.NewPasswordEntry()
	cw.PasswordEntry.SetPlaceHolder("Password (optional)")

	// Connect form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Network ID", Widget: cw.NetworkIDEntry},
			{Text: "Password", Widget: cw.PasswordEntry},
		},
		SubmitText: "Connect",
		OnSubmit:   cw.connect,
		OnCancel: func() {
			cw.Close()
		},
	}

	// Main container
	cw.Container = container.NewVBox(
		widget.NewLabel("Connect to an existing goVPN network"),
		form,
	)

	return cw.Container
}

// connect attempts to join the specified network
func (cw *ConnectWindow) connect() {
	networkID := cw.NetworkIDEntry.Text
	// A senha seria utilizada aqui para conectar à rede,
	// mas não estamos implementando isso agora.
	// Na implementação completa:
	// err := cw.UI.VPN.NetworkManager.JoinNetwork(networkID, cw.PasswordEntry.Text)

	// For now, we'll just show a message and hide the window
	cw.UI.ShowMessage("Connection", "Attempting to connect to network: "+networkID)

	// Close the window after attempt
	cw.Close()
}

// Show sobrescreve o método Show da BaseWindow para garantir que o conteúdo seja criado corretamente
func (cw *ConnectWindow) Show() {
	// Se a janela foi destruída, cria uma nova
	if cw.Window == nil {
		cw.Window = cw.UI.createWindow(cw.Title, cw.Width, cw.Height, cw.Resizable)
		// Adiciona novamente o manipulador para quando a janela for fechada
		cw.Window.SetOnClosed(func() {
			cw.Window = nil
			// Também limpa a referência no UIManager quando a janela é fechada pelo "X"
			cw.UI.ConnectWindow = nil
		})
		// Sempre recria o conteúdo para evitar problemas com referências antigas
		cw.Content = nil
	}

	// Cria o conteúdo - sempre recria para evitar problemas
	cw.Content = cw.CreateContent()

	// Define o conteúdo da janela
	if cw.Content != nil {
		cw.Window.SetContent(cw.Content)
	} else {
		// Se o conteúdo for nulo, exibe um erro
		errorLabel := widget.NewLabel("Erro: Não foi possível criar o conteúdo da janela")
		closeButton := widget.NewButton("Fechar", func() {
			cw.Close()
		})

		errorContent := container.NewCenter(
			container.NewVBox(
				errorLabel,
				closeButton,
			),
		)

		cw.Window.SetContent(errorContent)
	}

	// Exibe a janela centralizada
	cw.Window.CenterOnScreen()
	cw.Window.Show()
}

// Close sobrescreve o método Close da BaseWindow para garantir que a referência no UIManager seja limpa
func (cw *ConnectWindow) Close() {
	// Chama o método Close da classe pai
	cw.BaseWindow.Close()

	// Limpa a referência no UIManager
	cw.UI.ConnectWindow = nil
}
