package main

import (
	"image/color"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// AboutWindow manages the "About" interface
type AboutWindow struct {
	UI        *UIManager
	Window    fyne.Window
	Container *fyne.Container
}

// NewAboutWindow creates a new "About" window
func NewAboutWindow(ui *UIManager) *AboutWindow {
	aboutWindow := &AboutWindow{
		UI:     ui,
		Window: ui.createWindow("About - goVPN", 400, 300, false),
	}

	// Add handler for when the window is closed
	aboutWindow.Window.SetOnClosed(func() {
		aboutWindow.Window = nil
	})

	return aboutWindow
}

// Show displays the "About" window
func (aw *AboutWindow) Show() {
	// If the window has been destroyed, create a new one
	if aw.Window == nil {
		aw.Window = aw.UI.createWindow("About - goVPN", 400, 300, false)
		// Re-add the handler for when the window is closed
		aw.Window.SetOnClosed(func() {
			aw.Window = nil
		})
	}

	// Initialize necessary components before showing the window
	content := aw.CreateContent()

	// Set the window content
	aw.Window.SetContent(content)

	// Display the window centered
	aw.Window.CenterOnScreen()
	aw.Window.Show()
}

// CreateContent creates the content for the "About" window
func (aw *AboutWindow) CreateContent() fyne.CanvasObject {
	// Title
	title := widget.NewLabelWithStyle("goVPN", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Version
	version := widget.NewLabelWithStyle("Version 1.0.0", fyne.TextAlignCenter, fyne.TextStyle{})

	// Description
	description := widget.NewLabelWithStyle(
		"Peer-to-peer (P2P) VPN client using WebRTC for secure communication.",
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)
	description.Wrapping = fyne.TextWrapWord

	// Author
	authors := widget.NewLabelWithStyle(
		"Developed by: goVPN Team",
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)

	// Social links
	githubURL, _ := url.Parse("https://github.com/itxtoledo/goVPN")
	twitterURL, _ := url.Parse("https://x.com/itxtoledo")

	githubLink := widget.NewHyperlink("GitHub: github.com/itxtoledo/goVPN", githubURL)
	githubLink.Alignment = fyne.TextAlignCenter

	twitterLink := widget.NewHyperlink("X: @itxtoledo", twitterURL)
	twitterLink.Alignment = fyne.TextAlignCenter

	linksContainer := container.NewVBox(
		githubLink,
		twitterLink,
	)

	// Logo (colored text as a placeholder for an image)
	logo := canvas.NewText("goVPN", color.NRGBA{R: 0, G: 180, B: 100, A: 255})
	logo.TextSize = 48
	logo.Alignment = fyne.TextAlignCenter

	// Close button
	closeButton := widget.NewButton("Close", func() {
		if aw.Window != nil {
			// Completely close the window instead of just hiding it
			aw.Window.Close()
		}
	})

	// Main container
	content := container.NewVBox(
		logo,
		title,
		version,
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		authors,
		widget.NewSeparator(),
		linksContainer,
		closeButton,
	)

	return content
}
