// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/about_tab_component.go
package main

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// AboutTabComponent represents the About tab component
type AboutTabComponent struct {
	UI        *UIManager
	Container *fyne.Container
}

// NewAboutTabComponent creates a new instance of the About tab component
func NewAboutTabComponent(ui *UIManager) *AboutTabComponent {
	comp := &AboutTabComponent{
		UI: ui,
	}

	comp.createContent()
	return comp
}

// createContent creates the content for the About tab
func (a *AboutTabComponent) createContent() {
	// About Tab - contains the application information
	repoLink := widget.NewHyperlink("GitHub: github.com/itxtoledo/goVPN", a.parseURL("https://github.com/itxtoledo/goVPN"))
	xLink := widget.NewHyperlink("X: @itxtoledo", a.parseURL("https://x.com/itxtoledo"))

	a.Container = container.NewVBox(
		widget.NewLabelWithStyle("goVPN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Version 1.0.0"),
		widget.NewSeparator(),
		widget.NewLabel(
			"Peer-to-peer (P2P) VPN client using WebRTC for secure communication.",
		),
		widget.NewSeparator(),
		widget.NewLabel(
			"Developed by: goVPN Team",
		),
		widget.NewSeparator(),
		repoLink,
		xLink,
	)
}

// parseURL converts a URL string to a URL object
func (a *AboutTabComponent) parseURL(urlStr string) *url.URL {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}
	return parsedURL
}

// GetContainer returns the About tab container
func (a *AboutTabComponent) GetContainer() *fyne.Container {
	return a.Container
}
