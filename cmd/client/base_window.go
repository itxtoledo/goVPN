package main

import (
	"fyne.io/fyne/v2"
)

// BaseWindow representa uma janela base para outras janelas do aplicativo
type BaseWindow struct {
	Window       fyne.Window
	Content      *fyne.Container
	Title        string
	createWindow func(title string, width, height float32) fyne.Window
	width        float32
	height       float32
}

// NewBaseWindow cria uma nova janela base
func NewBaseWindow(createWindow func(title string, width, height float32) fyne.Window, title string, width, height float32) *BaseWindow {
	bw := &BaseWindow{
		Title:        title,
		createWindow: createWindow,
		width:        width,
		height:       height,
	}

	// Criar a janela
	bw.Window = createWindow(title, width, height)

	// Configurar callback de fechamento
	bw.Window.SetOnClosed(func() {
		// Limpar referências mas manter a função de criação
		bw.Window = nil
	})

	return bw
}

// Show exibe a janela
func (bw *BaseWindow) Show() {
	if bw.Window == nil {
		// Recriar a janela se ela foi fechada
		bw.Window = bw.createWindow(bw.Title, bw.width, bw.height)

		// Reconfigurar o callback de fechamento
		bw.Window.SetOnClosed(func() {
			bw.Window = nil
		})

		// Restaurar o conteúdo se existir
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
