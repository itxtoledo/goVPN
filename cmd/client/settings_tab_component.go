package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/data"
)

// SettingsTabComponent representa o componente da aba de configurações
type SettingsTabComponent struct {
	ComputerNameEntry  *widget.Entry
	ServerAddressEntry *widget.Entry
	ThemeSelect        *widget.Select
	SaveButton         *widget.Button

	// Dependencies
	ConfigManager *ConfigManager
	RealtimeData  *data.RealtimeDataLayer
	App           fyne.App
	refreshUI     func()
}

// NewSettingsTabComponent cria uma nova instância do componente de configurações
func NewSettingsTabComponent(configManager *ConfigManager, realtimeData *data.RealtimeDataLayer, app fyne.App, refreshUI func()) *SettingsTabComponent {
	stc := &SettingsTabComponent{
		ConfigManager: configManager,
		RealtimeData:  realtimeData,
		App:           app,
		refreshUI:     refreshUI,
	}
	return stc
}

// CreateSettingsContainer cria o container com os controles de configuração
func (stc *SettingsTabComponent) CreateSettingsContainer() *fyne.Container {
	// Obter as configurações atuais
	config := stc.ConfigManager.GetConfig()

	// Campo de entrada para o nome de usuário
	stc.ComputerNameEntry = widget.NewEntry()
	stc.ComputerNameEntry.SetText(config.ComputerName)
	stc.ComputerNameEntry.SetPlaceHolder("Enter your computername")

	// Campo de entrada para o endereço do servidor
	stc.ServerAddressEntry = widget.NewEntry()
	stc.ServerAddressEntry.SetText(config.ServerAddress)
	stc.ServerAddressEntry.SetPlaceHolder("Enter server address (ws://host:port)")

	// Seletor de tema
	themeOptions := []string{"System", "Light", "Dark"}
	stc.ThemeSelect = widget.NewSelect(themeOptions, func(selected string) {
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
	stc.ThemeSelect.SetSelectedIndex(themeIndex)

	// Botão Salvar
	stc.SaveButton = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		stc.saveSettings()
	})

	// Formulário de configurações
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "ComputerName", Widget: stc.ComputerNameEntry, HintText: "Your display name in the VPN"},
			{Text: "Server", Widget: stc.ServerAddressEntry, HintText: "Address of the signaling server"},
			{Text: "Theme", Widget: stc.ThemeSelect, HintText: "Application theme"},
		},
	}

	// Container de botões
	buttonContainer := container.NewHBox(
		layout.NewSpacer(),
		stc.SaveButton,
	)

	// Container principal
	content := container.NewBorder(
		nil,
		buttonContainer,
		nil,
		nil,
		form,
	)

	return container.NewPadded(content)
}

// saveSettings salva as configurações
func (stc *SettingsTabComponent) saveSettings() {
	// Obter as configurações atuais
	config := stc.ConfigManager.GetConfig()

	// Atualizar com os novos valores
	config.ComputerName = stc.ComputerNameEntry.Text
	config.ServerAddress = stc.ServerAddressEntry.Text

	// Atualizar o tema
	switch stc.ThemeSelect.SelectedIndex() {
	case 0:
		config.Theme = "system"
	case 1:
		config.Theme = "light"
	case 2:
		config.Theme = "dark"
	}

	// Salvar as novas configurações
	err := stc.ConfigManager.UpdateConfig(config)
	if err != nil {
		log.Printf("Error saving settings: %v", err)
	}

	// Aplicar as configurações
	stc.applySettings(config)
}

// applySettings aplica as configurações
func (stc *SettingsTabComponent) applySettings(config Config) {
	// Atualizar o tema
	switch config.Theme {
	case "light":
		stc.App.Settings().SetTheme(theme.LightTheme())
	case "dark":
		stc.App.Settings().SetTheme(theme.DarkTheme())
	default:
		stc.App.Settings().SetTheme(stc.App.Settings().Theme())
	}

	// Atualizar o nome de usuário na camada de dados em tempo real
	stc.RealtimeData.SetComputerName(config.ComputerName)

	// Atualizar o endereço do servidor
	stc.RealtimeData.SetServerAddress(config.ServerAddress)

	// Emitir evento de configurações alteradas
	stc.RealtimeData.EmitEvent(data.EventSettingsChanged, "Settings updated", nil)

	// Atualizar a UI
	stc.refreshUI()
}
