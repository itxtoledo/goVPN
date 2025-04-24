package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
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

	// Ensure content is created immediately after window initialization
	settingsWindow.Content = settingsWindow.CreateContent()

	return settingsWindow
}

// CreateContent creates the content for the settings window
func (sw *SettingsWindow) CreateContent() fyne.CanvasObject {
	// Load current settings
	config := sw.UI.ConfigManager.GetConfig()

	// Server settings
	sw.ServerEntry = widget.NewEntry()
	sw.ServerEntry.SetPlaceHolder("Signaling server address")
	sw.ServerEntry.SetText(config.SignalServer)

	// Behavior settings
	sw.AutoConnectCheck = widget.NewCheck("Auto-connect to last network", nil)
	sw.AutoConnectCheck.Checked = config.AutoConnect

	sw.StartupCheck = widget.NewCheck("Start on system startup", nil)
	sw.StartupCheck.Checked = config.StartOnSystemBoot

	sw.NotificationsCheck = widget.NewCheck("Enable notifications", nil)
	sw.NotificationsCheck.Checked = config.EnableNotifications

	// Appearance settings
	sw.ThemeSelect = widget.NewSelect([]string{"System Default", "Light", "Dark"}, nil)
	sw.ThemeSelect.SetSelected(config.ThemePreference)

	sw.LanguageSelect = widget.NewSelect([]string{"English", "Spanish"}, nil)
	sw.LanguageSelect.SetSelected(config.Language)

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
	// Get current settings
	config := sw.UI.ConfigManager.GetConfig()

	// Update settings with interface values
	config.SignalServer = sw.ServerEntry.Text
	config.ThemePreference = sw.ThemeSelect.Selected
	config.Language = sw.LanguageSelect.Selected
	config.AutoConnect = sw.AutoConnectCheck.Checked
	config.StartOnSystemBoot = sw.StartupCheck.Checked
	config.EnableNotifications = sw.NotificationsCheck.Checked

	// Save settings
	sw.UI.ConfigManager.UpdateConfig(config)

	// Update signal server in NetworkManager
	sw.UI.VPN.NetworkManager.SignalServer = config.SignalServer

	// Apply theme
	switch config.ThemePreference {
	case "Light":
		sw.UI.App.Settings().SetTheme(theme.LightTheme())
	case "Dark":
		sw.UI.App.Settings().SetTheme(theme.DarkTheme())
	case "System Default":
		sw.UI.App.Settings().SetTheme(theme.DefaultTheme())
	}

	// Apply other settings-based adjustments
	// (Code for system boot startup, etc. would be implemented here)

	// Close the window
	sw.Close()
}

// Show overrides the BaseWindow Show method to ensure content is created correctly
func (sw *SettingsWindow) Show() {
	// If window was destroyed, create a new one
	if sw.Window == nil {
		sw.Window = sw.UI.createWindow(sw.Title, sw.Width, sw.Height, sw.Resizable)
		// Add handler for when window is closed
		sw.Window.SetOnClosed(func() {
			sw.Window = nil
			// Also clear reference in UIManager when window is closed by "X"
			sw.UI.SettingsWindow = nil
		})
		// Always recreate content to avoid problems with old references
		sw.Content = nil
	}

	// Create content - always recreate to avoid issues
	sw.Content = sw.CreateContent()

	// Set window content
	if sw.Content != nil {
		sw.Window.SetContent(sw.Content)
	} else {
		// If content is null, display an error
		errorLabel := widget.NewLabel("Error: Could not create window content")
		closeButton := widget.NewButton("Close", func() {
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

	// Display centered window
	sw.Window.CenterOnScreen()
	sw.Window.Show()
}

// Close overrides the BaseWindow Close method to ensure reference in UIManager is cleared
func (sw *SettingsWindow) Close() {
	// Call parent class Close method
	sw.BaseWindow.Close()

	// Clear reference in UIManager
	sw.UI.SettingsWindow = nil
}
