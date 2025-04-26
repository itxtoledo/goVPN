package main

import (
	"log"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// SettingsWindow representa a janela de configurações do aplicativo
type SettingsWindow struct {
	*BaseWindow
	UsernameEntry      *widget.Entry
	ServerAddressEntry *widget.Entry
	ThemeSelect        *widget.Select
	SaveButton         *widget.Button
	CancelButton       *widget.Button
}

// NewSettingsWindow cria uma nova janela de configurações
func NewSettingsWindow(ui *UIManager) *SettingsWindow {
	sw := &SettingsWindow{
		BaseWindow: NewBaseWindow(ui, "Settings", 400, 300),
	}

	// Obter as configurações atuais
	config := ui.ConfigManager.GetConfig()

	// Campo de entrada para o nome de usuário
	sw.UsernameEntry = widget.NewEntry()
	sw.UsernameEntry.SetText(config.Username)
	sw.UsernameEntry.SetPlaceHolder("Enter your username")

	// Campo de entrada para o endereço do servidor
	sw.ServerAddressEntry = widget.NewEntry()
	sw.ServerAddressEntry.SetText(config.ServerAddress)
	sw.ServerAddressEntry.SetPlaceHolder("Enter server address (ws://host:port)")

	// Seletor de tema
	themeOptions := []string{"System", "Light", "Dark"}
	sw.ThemeSelect = widget.NewSelect(themeOptions, func(selected string) {
		// Nada aqui, a mudança só é aplicada quando Save é clicado
	})

	// Selecionar o tema atual
	themeIndex := 0 // System por padrão
	switch config.Theme {
	case "light":
		themeIndex = 1
	case "dark":
		themeIndex = 2
	}
	sw.ThemeSelect.SetSelectedIndex(themeIndex)

	// Botão Salvar
	sw.SaveButton = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		sw.saveSettings()
	})

	// Botão Cancelar
	sw.CancelButton = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		sw.Close()
	})

	// Container para os botões
	buttonContainer := container.NewHBox(
		layout.NewSpacer(),
		sw.CancelButton,
		sw.SaveButton,
	)

	// Formulário de configurações
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Username", Widget: sw.UsernameEntry, HintText: "Your display name in the VPN"},
			{Text: "Server Address", Widget: sw.ServerAddressEntry, HintText: "Address of the signaling server"},
			{Text: "Theme", Widget: sw.ThemeSelect, HintText: "Application theme"},
		},
	}

	// Container principal com o formulário e botões
	content := container.NewBorder(
		nil,
		buttonContainer,
		nil,
		nil,
		form,
	)

	// Adicionar padding
	paddedContent := container.NewPadded(content)

	// Definir o conteúdo da janela
	sw.SetContent(paddedContent)

	return sw
}

// saveSettings salva as configurações
func (sw *SettingsWindow) saveSettings() {
	// Obter as configurações atuais
	config := sw.UI.ConfigManager.GetConfig()

	// Atualizar com os novos valores
	config.Username = sw.UsernameEntry.Text
	config.ServerAddress = sw.ServerAddressEntry.Text

	// Atualizar o tema
	switch sw.ThemeSelect.SelectedIndex() {
	case 0:
		config.Theme = "system"
	case 1:
		config.Theme = "light"
	case 2:
		config.Theme = "dark"
	}

	// Salvar as novas configurações
	err := sw.UI.ConfigManager.UpdateConfig(config)
	if err != nil {
		log.Printf("Error saving settings: %v", err)
	}

	// Aplicar as configurações
	sw.applySettings(config)

	// Fechar a janela
	sw.Close()
}

// applySettings aplica as configurações
func (sw *SettingsWindow) applySettings(config storage.Config) {
	// Atualizar o tema
	switch config.Theme {
	case "light":
		sw.UI.App.Settings().SetTheme(theme.LightTheme())
	case "dark":
		sw.UI.App.Settings().SetTheme(theme.DarkTheme())
	default:
		// System é o padrão
	}

	// Atualizar o nome de usuário na camada de dados em tempo real
	sw.UI.RealtimeData.SetUsername(config.Username)

	// Emitir evento de configurações alteradas
	sw.UI.RealtimeData.EmitEvent(data.EventSettingsChanged, "Settings updated", nil)

	// Atualizar a UI
	sw.UI.refreshUI()
}
