package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// AboutWindow gerencia a interface "Sobre"
type AboutWindow struct {
	UI        *UIManager
	Window    fyne.Window
	Container *fyne.Container
}

// NewAboutWindow cria uma nova janela "Sobre"
func NewAboutWindow(ui *UIManager) *AboutWindow {
	aboutWindow := &AboutWindow{
		UI:     ui,
		Window: ui.createWindow("Sobre - goVPN", 400, 300, false),
	}

	return aboutWindow
}

// Show exibe a janela "Sobre"
func (aw *AboutWindow) Show() {
	// Se a janela já foi destruída, cria uma nova
	if aw.Window == nil {
		aw.Window = aw.UI.createWindow("Sobre - goVPN", 400, 300, false)
	}

	// Inicializa os componentes necessários antes de exibir a janela
	content := aw.CreateContent()

	// Define o conteúdo da janela
	aw.Window.SetContent(content)

	// Exibe a janela centralizada
	aw.Window.CenterOnScreen()
	aw.Window.Show()
}

// CreateContent cria o conteúdo da janela "Sobre"
func (aw *AboutWindow) CreateContent() fyne.CanvasObject {
	// Título
	title := widget.NewLabelWithStyle("goVPN", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Versão
	version := widget.NewLabelWithStyle("Versão 1.0.0", fyne.TextAlignCenter, fyne.TextStyle{})

	// Descrição
	description := widget.NewLabelWithStyle(
		"Cliente de VPN ponto a ponto (P2P) utilizando WebRTC para comunicação segura.",
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)
	description.Wrapping = fyne.TextWrapWord

	// Autor
	authors := widget.NewLabelWithStyle(
		"Desenvolvido por: Equipe goVPN",
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)

	// Logo (texto colorido como placeholder para uma imagem)
	logo := canvas.NewText("goVPN", color.NRGBA{R: 0, G: 180, B: 100, A: 255})
	logo.TextSize = 48
	logo.Alignment = fyne.TextAlignCenter

	// Botão de fechar
	closeButton := widget.NewButton("Fechar", func() {
		if aw.Window != nil {
			// Fechar a janela completamente em vez de apenas escondê-la
			aw.Window.Close()
		}
	})

	// Container principal
	content := container.NewVBox(
		logo,
		title,
		version,
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		authors,
		closeButton,
	)

	return content
}
