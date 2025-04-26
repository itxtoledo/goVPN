package main

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/icon"
)

// AboutWindow representa a janela de informações sobre o aplicativo
type AboutWindow struct {
	*BaseWindow
}

// NewAboutWindow cria uma nova janela de informações sobre o aplicativo
func NewAboutWindow(ui *UIManager) *AboutWindow {
	aw := &AboutWindow{
		BaseWindow: NewBaseWindow(ui, "About "+AppTitleName, 400, 400),
	}

	// Configurar o conteúdo da janela
	logo := canvas.NewImageFromResource(icon.VPN)
	logo.SetMinSize(fyne.NewSize(128, 128))

	titleLabel := widget.NewLabelWithStyle(
		AppTitleName,
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)
	titleLabel.TextStyle.Bold = true

	versionLabel := widget.NewLabelWithStyle(
		"Version "+AppVersion,
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)

	descriptionLabel := widget.NewLabelWithStyle(
		AppDescription,
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)

	authorLabel := widget.NewLabelWithStyle(
		"Created by "+AppAuthor,
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)

	// Criar um link para o repositório
	repoURL, _ := url.Parse(AppRepository)
	repoLink := widget.NewHyperlink("Source Code", repoURL)
	repoContainer := container.New(layout.NewCenterLayout(), repoLink)

	// Criar um botão OK para fechar a janela
	okButton := widget.NewButton("OK", func() {
		aw.Close()
	})
	okContainer := container.New(layout.NewCenterLayout(), okButton)

	// Criar o conteúdo principal
	content := container.NewVBox(
		container.New(layout.NewCenterLayout(), logo),
		titleLabel,
		versionLabel,
		widget.NewSeparator(),
		descriptionLabel,
		authorLabel,
		widget.NewSeparator(),
		repoContainer,
		layout.NewSpacer(),
		okContainer,
	)

	// Adicionar padding
	paddedContent := container.NewPadded(content)

	// Definir o conteúdo da janela
	aw.SetContent(paddedContent)

	return aw
}
