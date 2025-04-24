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
	*BaseWindow
}

// NewAboutWindow creates a new "About" window
func NewAboutWindow(ui *UIManager) *AboutWindow {
	aboutWindow := &AboutWindow{
		BaseWindow: NewBaseWindow(ui, "About - "+AppTitleName, 400, 300, false),
	}

	// Garantir que o conteúdo é criado imediatamente após a inicialização da janela
	aboutWindow.Content = aboutWindow.CreateContent()

	return aboutWindow
}

// CreateContent creates the content for the "About" window
func (aw *AboutWindow) CreateContent() fyne.CanvasObject {
	// Title
	title := widget.NewLabelWithStyle(AppTitleName, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

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
		"Developed by: "+AppTitleName+" Team",
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)

	// Made in Brazil by Gustavo Toledo
	madeInBrazil := widget.NewLabelWithStyle(
		"Proudly made in Brazil by Gustavo Toledo",
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
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
	logo := canvas.NewText(AppTitleName, color.NRGBA{R: 0, G: 180, B: 100, A: 255})
	logo.TextSize = 48
	logo.Alignment = fyne.TextAlignCenter

	// Close button
	closeButton := widget.NewButton("Close", func() {
		aw.Close()
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
		madeInBrazil,
		widget.NewSeparator(),
		linksContainer,
		closeButton,
	)

	return content
}

// Show sobrescreve o método Show da BaseWindow para garantir que o conteúdo seja criado corretamente
func (aw *AboutWindow) Show() {
	// Se a janela foi destruída, cria uma nova
	if aw.Window == nil {
		aw.Window = aw.UI.createWindow(aw.Title, aw.Width, aw.Height, aw.Resizable)
		// Adiciona novamente o manipulador para quando a janela for fechada
		aw.Window.SetOnClosed(func() {
			aw.Window = nil
			// Também limpa a referência no UIManager quando a janela é fechada pelo "X"
			aw.UI.AboutWindow = nil
		})
		// Sempre recria o conteúdo para evitar problemas com referências antigas
		aw.Content = nil
	}

	// Cria o conteúdo - sempre recria para evitar problemas
	aw.Content = aw.CreateContent()

	// Define o conteúdo da janela
	if aw.Content != nil {
		aw.Window.SetContent(aw.Content)
	} else {
		// Se o conteúdo for nulo, exibe um erro
		errorLabel := widget.NewLabel("Erro: Não foi possível criar o conteúdo da janela")
		closeButton := widget.NewButton("Fechar", func() {
			aw.Close()
		})

		errorContent := container.NewCenter(
			container.NewVBox(
				errorLabel,
				closeButton,
			),
		)

		aw.Window.SetContent(errorContent)
	}

	// Exibe a janela centralizada
	aw.Window.CenterOnScreen()
	aw.Window.Show()
}

// Close sobrescreve o método Close da BaseWindow para garantir que a referência no UIManager seja limpa
func (aw *AboutWindow) Close() {
	// Chama o método Close da classe pai
	aw.BaseWindow.Close()

	// Limpa a referência no UIManager
	aw.UI.AboutWindow = nil
}
