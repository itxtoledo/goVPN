package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/storage"
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

	// Callback
	OnSettingsSaved func(config storage.Config)
}

// NewSettingsWindow creates a new settings window
func NewSettingsWindow(app fyne.App, currentConfig storage.Config, onSettingsSaved func(config storage.Config)) *SettingsWindow {
	if globalSettingsWindow != nil {
		return globalSettingsWindow
	}

	sw := &SettingsWindow{
		BaseWindow:      NewBaseWindow(app, "Settings", 320, 260),
		OnSettingsSaved: onSettingsSaved,
	}

	// Set close callback to reset the global instance when window closes
	sw.BaseWindow.Window.SetOnClosed(func() {
		globalSettingsWindow = nil
	})

	globalSettingsWindow = sw

	// Computer Name Entry
	sw.ComputerNameEntry = widget.NewEntry()
	sw.ComputerNameEntry.SetText(currentConfig.ComputerName)
	sw.ComputerNameEntry.SetPlaceHolder("Enter your computername")

	// Server Address Entry
	sw.ServerAddressEntry = widget.NewEntry()
	sw.ServerAddressEntry.SetText(currentConfig.ServerAddress)
	sw.ServerAddressEntry.SetPlaceHolder("Enter server address (ws://host:port)")

	// Theme Selector
	themeOptions := []string{"System", "Light", "Dark"}
	sw.ThemeSelect = widget.NewSelect(themeOptions, func(selected string) {
		// Change is applied only when Save is clicked
	})

	// Select current theme
	themeIndex := 0 // System by default
	switch currentConfig.Theme {
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

	return sw
}

// saveSettings saves the settings
func (sw *SettingsWindow) saveSettings() {
	// Create a new config object with updated values
	newConfig := storage.Config{
		ComputerName:  sw.ComputerNameEntry.Text,
		ServerAddress: sw.ServerAddressEntry.Text,
	}

	// Update theme
	switch sw.ThemeSelect.SelectedIndex() {
	case 0:
		newConfig.Theme = "system"
	case 1:
		newConfig.Theme = "light"
	case 2:
		newConfig.Theme = "dark"
	}

	// Invoke the callback with the new config
	sw.OnSettingsSaved(newConfig)
}

// Show displays the settings window
func (sw *SettingsWindow) Show() {
	// Create title with icon
	titleIcon := widget.NewIcon(theme.SettingsIcon())
	titleLabel := widget.NewLabel("Settings")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleContainer := container.NewHBox(titleIcon, titleLabel)

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
	content := container.NewVBox(
		container.NewPadded(titleContainer),
		widget.NewSeparator(),
		container.NewPadded(form),
		widget.NewSeparator(),
		container.NewPadded(buttonContainer),
	)

	sw.SetContent(content)
	sw.BaseWindow.Show()
}
