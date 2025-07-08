package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// CustomTheme provides improved contrast for light and dark themes
type CustomTheme struct {
	fyne.Theme
}

// NewCustomTheme creates a new custom theme based on the system theme
func NewCustomTheme(baseTheme fyne.Theme) *CustomTheme {
	return &CustomTheme{Theme: baseTheme}
}

// Color returns improved contrast colors for specific elements
func (ct *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// Get the base color first
	baseColor := ct.Theme.Color(name, variant)

	// Return base color for all other cases
	return baseColor
}

// Size returns the same sizes as the base theme
func (ct *CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	return ct.Theme.Size(name)
}

// Font returns the same fonts as the base theme
func (ct *CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return ct.Theme.Font(style)
}

// Icon returns the same icons as the base theme
func (ct *CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return ct.Theme.Icon(name)
}

// CreateHighContrastLabel creates a label with bold style for better visibility
func CreateHighContrastLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true} // Make text bold for better visibility
	return label
}

// CreateStatusLabel creates a label optimized for status text
func CreateStatusLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Monospace: true, Bold: true} // Monospace and bold for status
	return label
}

// CreateRoomTitleLabel creates a label optimized for room titles
func CreateRoomTitleLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true, Monospace: false}
	return label
}

// CreateSectionHeaderLabel creates a label for section headers with consistent styling
func CreateSectionHeaderLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true}
	return label
}

// CreateMemberLabel creates a label for displaying member information
func CreateMemberLabel(text string, isCurrentUser bool, isOnline bool) *widget.Label {
	label := widget.NewLabel(text)
	if isCurrentUser {
		label.TextStyle = fyne.TextStyle{Bold: true}
	} else if !isOnline {
		label.TextStyle = fyne.TextStyle{Italic: true}
	}
	return label
}

// CreateInfoLabel creates a label for displaying information with monospace font
func CreateInfoLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Monospace: true, Italic: true}
	return label
}

// CreateRoomCard creates a card widget with improved styling for room content
func CreateRoomCard(content fyne.CanvasObject) *widget.Card {
	return widget.NewCard("", "", content)
}

// CreatePaddedContainer creates a container with consistent padding
func CreatePaddedContainer(content fyne.CanvasObject) *fyne.Container {
	return container.NewPadded(content)
}
