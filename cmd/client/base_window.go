package main

import (
	"fyne.io/fyne/v2"
	"log"
)

// BaseWindow representa uma janela base para outras janelas do aplicativo
type BaseWindow struct {
	Window  fyne.Window
	Content *fyne.Container
	Title   string
}

// NewBaseWindow cria uma nova janela base
func NewBaseWindow(createWindow func(title string, width, height float32) fyne.Window, title string, width, height float32) *BaseWindow {
	bw := &BaseWindow{
		Title: title,
	}

	// Criar a janela
	bw.Window = createWindow(title, width, height)

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
		// This case should ideally not be reached if windows are managed correctly
		// However, if it is, we can't recreate it without the createWindow function.
		// For now, we'll just log an error.
		log.Println("Error: Attempted to show a closed window without a recreate function.")
		return
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
