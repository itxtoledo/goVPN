package main

import (
	"fyne.io/fyne/v2"
)

// BaseWindow representa uma janela base para outras janelas do aplicativo
type BaseWindow struct {
	UI      *UIManager
	Window  fyne.Window
	Content *fyne.Container
	Title   string
}

// NewBaseWindow cria uma nova janela base
func NewBaseWindow(ui *UIManager, title string, width, height float32) *BaseWindow {
	bw := &BaseWindow{
		UI:    ui,
		Title: title,
	}

	// Criar a janela
	bw.Window = ui.createWindow(title, width, height)

	// Configurar callback de fechamento
	bw.Window.SetOnClosed(func() {
		// Limpar referências
		bw.Window = nil
	})

	return bw
}

// Show exibe a janela
func (bw *BaseWindow) Show() {
	if bw.Window == nil {
		// Recriar a janela se foi fechada
		bw.Window = bw.UI.createWindow(bw.Title, 400, 300)

		// Configurar callback de fechamento
		bw.Window.SetOnClosed(func() {
			// Limpar referências
			bw.Window = nil
		})

		// Reconfigurar conteúdo
		if bw.Content != nil {
			bw.Window.SetContent(bw.Content)
		}
	}

	bw.Window.Show()
}

// Hide esconde a janela
func (bw *BaseWindow) Hide() {
	if bw.Window != nil {
		bw.Window.Hide()
	}
}

// Close fecha a janela
func (bw *BaseWindow) Close() {
	if bw.Window != nil {
		bw.Window.Close()
		bw.Window = nil
	}
}

// SetContent define o conteúdo da janela
func (bw *BaseWindow) SetContent(content *fyne.Container) {
	bw.Content = content

	if bw.Window != nil {
		bw.Window.SetContent(content)
	}
}
