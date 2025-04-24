package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// BaseWindow provides a base implementation for all application windows
type BaseWindow struct {
	UI        *UIManager
	Window    fyne.Window
	Title     string
	Width     int
	Height    int
	Resizable bool
	Content   fyne.CanvasObject // Armazenamos o conteúdo para reutilização
}

// NewBaseWindow creates a new base window
func NewBaseWindow(ui *UIManager, title string, width int, height int, resizable bool) *BaseWindow {
	baseWindow := &BaseWindow{
		UI:        ui,
		Title:     title,
		Width:     width,
		Height:    height,
		Resizable: resizable,
		Window:    ui.createWindow(title, width, height, resizable),
	}

	// Add handler for when the window is closed
	baseWindow.Window.SetOnClosed(func() {
		baseWindow.Window = nil
	})

	return baseWindow
}

// Show displays the window
func (bw *BaseWindow) Show() {
	// If the window has been destroyed, create a new one
	if bw.Window == nil {
		bw.Window = bw.UI.createWindow(bw.Title, bw.Width, bw.Height, bw.Resizable)
		// Re-add the handler for when the window is closed
		bw.Window.SetOnClosed(func() {
			bw.Window = nil
		})
		// O conteúdo também precisa ser recriado
		bw.Content = nil
	}

	// Cria o conteúdo apenas se ele ainda não foi criado
	if bw.Content == nil {
		bw.Content = bw.CreateContent()
	}

	// Set the window content if we have valid content
	if bw.Content != nil {
		bw.Window.SetContent(bw.Content)
	} else {
		// Se o conteúdo for nulo, exibe um erro em vez de travar
		errorLabel := widget.NewLabel("Erro: Não foi possível criar o conteúdo da janela")
		closeButton := widget.NewButton("Fechar", func() {
			bw.Close()
		})

		errorContent := container.New(layout.NewCenterLayout(),
			container.New(layout.NewVBoxLayout(),
				errorLabel,
				closeButton,
			),
		)

		bw.Window.SetContent(errorContent)
	}

	// Display the window centered
	bw.Window.CenterOnScreen()
	bw.Window.Show()
}

// CreateContent creates the content for the window - to be overridden by subclasses
func (bw *BaseWindow) CreateContent() fyne.CanvasObject {
	// This is a placeholder that should be overridden by subclasses
	return nil
}

// Hide hides the window
func (bw *BaseWindow) Hide() {
	if bw.Window != nil {
		bw.Window.Hide()
	}
}

// Close closes the window
func (bw *BaseWindow) Close() {
	if bw.Window != nil {
		bw.Window.Close()
		bw.Window = nil
		bw.Content = nil // Também limpa o conteúdo para que seja recriado na próxima abertura
	}
}
