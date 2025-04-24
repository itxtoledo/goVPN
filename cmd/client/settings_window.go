package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SettingsWindow represents the settings dialog
type SettingsWindow struct {
	*BaseWindow
	UI             *UIManager
	serverInput    *widget.Entry
	themeSelection *widget.Select
	usernameInput  *widget.Entry // New field for username input
}

// NewSettingsWindow creates a new settings window
func NewSettingsWindow(ui *UIManager) *SettingsWindow {
	sw := &SettingsWindow{
		BaseWindow: NewBaseWindow(ui, "Settings", 400, 300, true),
		UI:         ui,
	}

	// Initialize content immediately after window creation
	sw.Content = sw.CreateContent()

	return sw
}

// CreateContent builds the settings UI
func (s *SettingsWindow) CreateContent() fyne.CanvasObject {
	// Get current configuration
	config := s.UI.ConfigManager.GetConfig()

	// Create server input field
	serverLabel := widget.NewLabel("Signal Server URL:")
	s.serverInput = widget.NewEntry()
	s.serverInput.SetText(s.UI.VPN.NetworkManager.SignalServer)

	// Create theme selection dropdown
	themeLabel := widget.NewLabel("Theme:")
	s.themeSelection = widget.NewSelect([]string{"System Default", "Light", "Dark"}, func(selected string) {
		// Theme selection is handled on save
	})
	// Set current theme selection
	s.themeSelection.SetSelected(config.ThemePreference)

	// Create username input field
	usernameLabel := widget.NewLabel("Display Username:")
	s.usernameInput = widget.NewEntry()
	s.usernameInput.SetText(config.Username) // Load saved username from config
	s.usernameInput.SetPlaceHolder("Enter your display name")
	// Limitar nome de usuário a 10 caracteres
	s.usernameInput.Validator = func(s string) error {
		if len(s) > 10 {
			return fmt.Errorf("Username must be 10 characters or less")
		}
		return nil
	}

	// Create save and cancel buttons
	saveButton := widget.NewButton("Save", func() {
		s.saveSettings()
	})
	saveButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton("Cancel", func() {
		s.Close()
	})

	// Create button container
	buttonContainer := container.NewHBox(
		layout.NewSpacer(),
		cancelButton,
		saveButton,
	)

	// Create form layout
	form := container.NewVBox(
		serverLabel,
		s.serverInput,
		widget.NewSeparator(),
		themeLabel,
		s.themeSelection,
		widget.NewSeparator(),
		usernameLabel,
		s.usernameInput,
		layout.NewSpacer(),
	)

	// Main layout with padding
	content := container.NewPadded(
		container.NewBorder(
			nil,
			buttonContainer,
			nil,
			nil,
			form,
		),
	)

	return content
}

// Show sobrescreve o método Show da BaseWindow para garantir que o conteúdo seja criado corretamente
func (s *SettingsWindow) Show() {
	// Se a janela foi destruída, cria uma nova
	if s.Window == nil {
		s.Window = s.UI.createWindow(s.Title, s.Width, s.Height, s.Resizable)
		// Adiciona novamente o manipulador para quando a janela for fechada
		s.Window.SetOnClosed(func() {
			s.Window = nil
			// Também limpa a referência no UIManager quando a janela é fechada pelo "X"
			s.UI.SettingsWindow = nil
		})
		// Sempre recria o conteúdo para evitar problemas com referências antigas
		s.Content = nil
	}

	// Cria o conteúdo - sempre recria para evitar problemas
	s.Content = s.CreateContent()

	// Define o conteúdo da janela
	if s.Content != nil {
		s.Window.SetContent(s.Content)
	} else {
		// Se o conteúdo for nulo, exibe um erro
		errorLabel := widget.NewLabel("Erro: Não foi possível criar o conteúdo da janela")
		closeButton := widget.NewButton("Fechar", func() {
			s.Close()
		})

		errorContent := container.NewCenter(
			container.NewVBox(
				errorLabel,
				closeButton,
			),
		)

		s.Window.SetContent(errorContent)
	}

	// Exibe a janela centralizada
	s.Window.CenterOnScreen()
	s.Window.Show()
}

// Close sobrescreve o método Close da BaseWindow para garantir que a referência no UIManager seja limpa
func (s *SettingsWindow) Close() {
	// Chama o método Close da classe pai
	s.BaseWindow.Close()

	// Limpa a referência no UIManager
	s.UI.SettingsWindow = nil
}

// saveSettings saves the current settings
func (s *SettingsWindow) saveSettings() {
	config := s.UI.ConfigManager.GetConfig()

	// Save signal server
	newServer := s.serverInput.Text
	if newServer != s.UI.VPN.NetworkManager.SignalServer {
		s.UI.VPN.NetworkManager.SignalServer = newServer
		config.SignalServer = newServer
	}

	// Save theme preference
	themePreference := s.themeSelection.Selected
	if themePreference != config.ThemePreference {
		config.ThemePreference = themePreference

		// Apply theme change
		switch themePreference {
		case "Light":
			s.UI.App.Settings().SetTheme(theme.LightTheme())
		case "Dark":
			s.UI.App.Settings().SetTheme(theme.DarkTheme())
		default:
			s.UI.App.Settings().SetTheme(theme.DefaultTheme())
		}

		// Update colors of elements that depend on theme
		s.UI.UpdateThemeColors()
	}

	// Save username (truncando para 10 caracteres se necessário)
	username := s.usernameInput.Text
	if len(username) > 10 {
		username = username[:10]
	}

	if username != config.Username {
		config.Username = username
		// Update the NetworkManager username
		if s.UI.VPN.NetworkManager != nil {
			s.UI.VPN.NetworkManager.Username = username
		}
	}

	// Save to file
	s.UI.ConfigManager.UpdateConfig(config)

	s.Close()
}
