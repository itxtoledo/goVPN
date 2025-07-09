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

// Global variable to ensure only one settings window can be open
var globalSettingsWindow *SettingsWindow

// SettingsWindow represents the settings window
type SettingsWindow struct {
	*BaseWindow
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

// NewSettingsWindow creates a new settings window
func NewSettingsWindow(ui *UIManager, configManager *ConfigManager, realtimeData *data.RealtimeDataLayer, app fyne.App, refreshUI func()) *SettingsWindow {
	if globalSettingsWindow != nil {
		return globalSettingsWindow
	}

	sw := &SettingsWindow{
		BaseWindow:    NewBaseWindow(ui.createWindow, "Settings", 600, 400),
		ConfigManager: configManager,
		RealtimeData:  realtimeData,
		App:           app,
		refreshUI:     refreshUI,
	}

	// Set close callback to reset the global instance when window closes
	sw.BaseWindow.Window.SetOnClosed(func() {
		globalSettingsWindow = nil
	})

	globalSettingsWindow = sw

	// Get current settings
	config := sw.ConfigManager.GetConfig()

	// Computer Name Entry
	sw.ComputerNameEntry = widget.NewEntry()
	sw.ComputerNameEntry.SetText(config.ComputerName)
	sw.ComputerNameEntry.SetPlaceHolder("Enter your computername")

	// Server Address Entry
	sw.ServerAddressEntry = widget.NewEntry()
	sw.ServerAddressEntry.SetText(config.ServerAddress)
	sw.ServerAddressEntry.SetPlaceHolder("Enter server address (ws://host:port)")

	// Theme Selector
	themeOptions := []string{"System", "Light", "Dark"}
	sw.ThemeSelect = widget.NewSelect(themeOptions, func(selected string) {
		// Change is applied only when Save is clicked
	})

	// Select current theme
	themeIndex := 0 // System by default
	switch config.Theme {
	case "light":
		themeIndex = 1
	case "dark":
		themeIndex = 2
	}
	sw.ThemeSelect.SetSelectedIndex(themeIndex)

	// Save Button
	sw.SaveButton = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		sw.saveSettings()
	})

	// Settings Form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "ComputerName", Widget: sw.ComputerNameEntry, HintText: "Your display name in the VPN"},
			{Text: "Server", Widget: sw.ServerAddressEntry, HintText: "Address of the signaling server"},
			{Text: "Theme", Widget: sw.ThemeSelect, HintText: "Application theme"},
		},
	}

	// Button Container
	buttonContainer := container.NewHBox(
		layout.NewSpacer(),
		sw.SaveButton,
	)

	// Main Container
	content := container.NewBorder(
		nil,
		buttonContainer,
		nil,
		nil,
		form,
	)

	// Add padding
	paddedContent := container.NewPadded(content)

	// Set window content
	sw.SetContent(paddedContent)

	return sw
}

// saveSettings saves the settings
func (sw *SettingsWindow) saveSettings() {
	// Get current settings
	config := sw.ConfigManager.GetConfig()

	// Update with new values
	config.ComputerName = sw.ComputerNameEntry.Text
	config.ServerAddress = sw.ServerAddressEntry.Text

	// Update theme
	switch sw.ThemeSelect.SelectedIndex() {
	case 0:
		config.Theme = "system"
	case 1:
		config.Theme = "light"
	case 2:
		config.Theme = "dark"
	}

	// Save new settings
	err := sw.ConfigManager.UpdateConfig(config)
	if err != nil {
		log.Printf("Error saving settings: %v", err)
	}

	// Apply settings
	sw.applySettings(config)
}

// applySettings applies the settings
func (sw *SettingsWindow) applySettings(config Config) {
	// Update theme
	switch config.Theme {
	case "light":
		sw.App.Settings().SetTheme(theme.LightTheme())
	case "dark":
		sw.App.Settings().SetTheme(theme.DarkTheme())
	default:
		sw.App.Settings().SetTheme(sw.App.Settings().Theme())
	}

	// Update computer name in realtime data layer
	sw.RealtimeData.SetComputerName(config.ComputerName)

	// Update server address
	sw.RealtimeData.SetServerAddress(config.ServerAddress)

	// Emit settings changed event
	sw.RealtimeData.EmitEvent(data.EventSettingsChanged, "Settings updated", nil)

	// Refresh UI
	sw.refreshUI()
}
