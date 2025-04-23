// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/about_tab_component.go
package main

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// AboutTabComponent representa o componente da aba Sobre
type AboutTabComponent struct {
	UI        *UIManager
	Container *fyne.Container
}

// NewAboutTabComponent cria uma nova instância do componente da aba Sobre
func NewAboutTabComponent(ui *UIManager) *AboutTabComponent {
	comp := &AboutTabComponent{
		UI: ui,
	}

	comp.createContent()
	return comp
}

// createContent cria o conteúdo da aba Sobre
func (a *AboutTabComponent) createContent() {
	// Tab Sobre - contém as informações da aplicação
	repoLink := widget.NewHyperlink("GitHub: github.com/itxtoledo/goVPN", a.parseURL("https://github.com/itxtoledo/goVPN"))
	xLink := widget.NewHyperlink("X: @itxtoledo", a.parseURL("https://x.com/itxtoledo"))

	a.Container = container.NewVBox(
		widget.NewLabelWithStyle("goVPN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Versão 1.0.0"),
		widget.NewSeparator(),
		widget.NewLabel(
			"Cliente de VPN ponto a ponto (P2P) utilizando WebRTC para comunicação segura.",
		),
		widget.NewSeparator(),
		widget.NewLabel(
			"Desenvolvido por: Equipe goVPN",
		),
		widget.NewSeparator(),
		repoLink,
		xLink,
	)
}

// parseURL converte uma string URL para um objeto URL
func (a *AboutTabComponent) parseURL(urlStr string) *url.URL {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}
	return parsedURL
}

// GetContainer retorna o container da aba Sobre
func (a *AboutTabComponent) GetContainer() *fyne.Container {
	return a.Container
}
