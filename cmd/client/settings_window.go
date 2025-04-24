package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// SettingsWindow manages the settings interface
type SettingsWindow struct {
	*BaseWindow
	ServerEntry        *widget.Entry
	AutoConnectCheck   *widget.Check
	StartupCheck       *widget.Check
	NotificationsCheck *widget.Check
	ThemeSelect        *widget.Select
	LanguageSelect     *widget.Select
	Container          *fyne.Container
}

// NewSettingsWindow creates a new settings window
func NewSettingsWindow(ui *UIManager) *SettingsWindow {
	settingsWindow := &SettingsWindow{
		BaseWindow: NewBaseWindow(ui, "Settings - goVPN", 500, 400, false),
	}

	// Garantir que o conteúdo é criado imediatamente após a inicialização da janela
	settingsWindow.Content = settingsWindow.CreateContent()

	return settingsWindow
}

// CreateContent creates the content for the settings window
func (sw *SettingsWindow) CreateContent() fyne.CanvasObject {
	// Server settings
	sw.ServerEntry = widget.NewEntry()
	sw.ServerEntry.SetPlaceHolder("Signaling server address")
	sw.ServerEntry.SetText(sw.UI.VPN.NetworkManager.SignalServer)

	// Behavior settings
	sw.AutoConnectCheck = widget.NewCheck("Auto-connect to last network", nil)
	sw.StartupCheck = widget.NewCheck("Start on system startup", nil)
	sw.NotificationsCheck = widget.NewCheck("Enable notifications", nil)

	// Appearance settings
	sw.ThemeSelect = widget.NewSelect([]string{"System Default", "Light", "Dark"}, nil)
	sw.ThemeSelect.Selected = "System Default"

	sw.LanguageSelect = widget.NewSelect([]string{"English", "Portuguese", "Spanish"}, nil)
	sw.LanguageSelect.Selected = "English"

	// Tabs for different settings categories
	tabs := container.NewAppTabs(
		container.NewTabItem("Connection", container.NewVBox(
			widget.NewLabel("Server Settings"),
			widget.NewForm(
				widget.NewFormItem("Signal Server", sw.ServerEntry),
			),
			sw.AutoConnectCheck,
		)),
		container.NewTabItem("Appearance", container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("Theme", sw.ThemeSelect),
				widget.NewFormItem("Language", sw.LanguageSelect),
			),
		)),
		container.NewTabItem("System", container.NewVBox(
			sw.StartupCheck,
			sw.NotificationsCheck,
		)),
	)

	// Button container
	buttons := container.NewHBox(
		widget.NewButton("Cancel", func() {
			sw.Close()
		}),
		widget.NewButton("Save", func() {
			sw.saveSettings()
		}),
	)

	// Main container
	sw.Container = container.NewBorder(
		nil,
		buttons,
		nil,
		nil,
		tabs,
	)

	return sw.Container
}

// saveSettings saves the current settings
func (sw *SettingsWindow) saveSettings() {
	// Update the settings
	sw.UI.VPN.NetworkManager.SignalServer = sw.ServerEntry.Text

	// Here we would save the other settings

	// Close the window
	sw.Close()
}

// Show sobrescreve o método Show da BaseWindow para garantir que o conteúdo seja criado corretamente
func (sw *SettingsWindow) Show() {
	// Se a janela foi destruída, cria uma nova
	if sw.Window == nil {
		sw.Window = sw.UI.createWindow(sw.Title, sw.Width, sw.Height, sw.Resizable)
		// Adiciona novamente o manipulador para quando a janela for fechada
		sw.Window.SetOnClosed(func() {
			sw.Window = nil
			// Também limpa a referência no UIManager quando a janela é fechada pelo "X"
			sw.UI.SettingsWindow = nil
		})
		// Sempre recria o conteúdo para evitar problemas com referências antigas
		sw.Content = nil
	}

	// Cria o conteúdo - sempre recria para evitar problemas
	sw.Content = sw.CreateContent()

	// Define o conteúdo da janela
	if sw.Content != nil {
		sw.Window.SetContent(sw.Content)
	} else {
		// Se o conteúdo for nulo, exibe um erro
		errorLabel := widget.NewLabel("Erro: Não foi possível criar o conteúdo da janela")
		closeButton := widget.NewButton("Fechar", func() {
			sw.Close()
		})

		errorContent := container.NewCenter(
			container.NewVBox(
				errorLabel,
				closeButton,
			),
		)

		sw.Window.SetContent(errorContent)
	}

	// Exibe a janela centralizada
	sw.Window.CenterOnScreen()
	sw.Window.Show()
}

// Close sobrescreve o método Close da BaseWindow para garantir que a referência no UIManager seja limpa
func (sw *SettingsWindow) Close() {
	// Chama o método Close da classe pai
	sw.BaseWindow.Close()

	// Limpa a referência no UIManager
	sw.UI.SettingsWindow = nil
}
