package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// RoomWindow manages the room (network) creation interface
type RoomWindow struct {
	*BaseWindow
	NameEntry     *widget.Entry
	PasswordEntry *widget.Entry
	Container     *fyne.Container
}

// NewRoomWindow creates a new room creation window
func NewRoomWindow(ui *UIManager) *RoomWindow {
	roomWindow := &RoomWindow{
		BaseWindow: NewBaseWindow(ui, "Create Network - goVPN", 350, 250, false),
	}

	return roomWindow
}

// CreateContent creates the content for the room window
func (rw *RoomWindow) CreateContent() fyne.CanvasObject {
	// Input fields - Criar novos campos a cada abertura da janela
	rw.NameEntry = widget.NewEntry()
	rw.NameEntry.SetPlaceHolder("Network Name")

	// Password field - restricted to 4 numeric digits
	rw.PasswordEntry = widget.NewPasswordEntry()
	rw.PasswordEntry.SetPlaceHolder("Password (4 digits)")
	rw.PasswordEntry.Validator = func(s string) error {
		if len(s) != 4 {
			return errors.New("Password must be exactly 4 digits")
		}
		matched, _ := regexp.MatchString("^[0-9]{4}$", s)
		if !matched {
			return errors.New("Password must contain only numbers")
		}
		return nil
	}

	// Creation form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: rw.NameEntry},
			{Text: "Password (4 digits)", Widget: rw.PasswordEntry},
		},
		SubmitText: "Create",
		OnSubmit:   rw.createNetwork,
		OnCancel: func() {
			rw.Close()
		},
	}

	// Main container
	rw.Container = container.NewVBox(
		widget.NewLabel("Create a new goVPN Network"),
		form,
	)

	return rw.Container
}

// Show sobrescreve o método Show da BaseWindow para garantir que o conteúdo seja criado corretamente
func (rw *RoomWindow) Show() {
	// Se a janela foi destruída, cria uma nova
	if rw.Window == nil {
		rw.Window = rw.UI.createWindow(rw.Title, rw.Width, rw.Height, rw.Resizable)
		// Adiciona novamente o manipulador para quando a janela for fechada
		rw.Window.SetOnClosed(func() {
			rw.Window = nil
			// Também limpa a referência no UIManager quando a janela é fechada pelo "X"
			rw.UI.RoomWindow = nil
		})
		// Sempre recria o conteúdo para evitar problemas com referências antigas
		rw.Content = nil
	}

	// Cria o conteúdo - sempre recria para evitar problemas
	rw.Content = rw.CreateContent()

	// Define o conteúdo da janela
	if rw.Content != nil {
		rw.Window.SetContent(rw.Content)
	} else {
		// Se o conteúdo for nulo, exibe um erro
		errorLabel := widget.NewLabel("Erro: Não foi possível criar o conteúdo da janela")
		closeButton := widget.NewButton("Fechar", func() {
			rw.Close()
		})

		errorContent := container.NewCenter(
			container.NewVBox(
				errorLabel,
				closeButton,
			),
		)

		rw.Window.SetContent(errorContent)
	}

	// Exibe a janela centralizada
	rw.Window.CenterOnScreen()
	rw.Window.Show()
}

// createNetwork attempts to create a new network
func (rw *RoomWindow) createNetwork() {
	name := rw.NameEntry.Text
	password := rw.PasswordEntry.Text

	// Validate inputs
	if name == "" {
		dialog.ShowError(errors.New("Network name cannot be empty"), rw.Window)
		return
	}

	// Validate password format
	if matched, _ := regexp.MatchString("^[0-9]{4}$", password); !matched {
		dialog.ShowError(errors.New("Password must be exactly 4 digits"), rw.Window)
		return
	}

	// Show progress message
	progressDialog := dialog.NewCustom("Creating Network", "Cancel", widget.NewLabel("Creating network, please wait..."), rw.Window)
	progressDialog.Show()

	// Create goroutine to avoid blocking UI
	go func() {
		// Actually send the create room command to the backend
		err := rw.UI.VPN.NetworkManager.CreateRoom(name, password)

		// Hide progress dialog
		progressDialog.Hide()

		if err != nil {
			// Show error on UI thread
			dialog.ShowError(fmt.Errorf("Failed to create network: %v", err), rw.Window)
			return
		}

		// Get the room ID that was created by the server
		roomID := rw.UI.VPN.CurrentRoom

		if roomID != "" {
			// Save the room to SQLite database if not already saved
			_, err := rw.UI.VPN.DB.Exec(
				"INSERT OR REPLACE INTO rooms (id, name, password, last_connected) VALUES (?, ?, ?, CURRENT_TIMESTAMP)",
				roomID, name, password,
			)
			if err != nil {
				log.Printf("Error saving room to database: %v", err)
			}

			// Success! Create dialog to show and copy the room ID
			roomIDEntry := widget.NewEntry()
			roomIDEntry.Text = roomID
			roomIDEntry.Disable()

			content := container.NewVBox(
				widget.NewLabel("Network created successfully!"),
				widget.NewLabel(""),
				widget.NewLabel("Network ID (share this with friends):"),
				roomIDEntry,
				widget.NewButton("Copy to Clipboard", func() {
					// Copy room ID to clipboard
					rw.Window.Clipboard().SetContent(roomID)
					dialog.ShowInformation("Copied", "Network ID copied to clipboard", rw.Window)
				}),
			)

			successDialog := dialog.NewCustom("Success", "Close", content, rw.Window)
			successDialog.Show()

			// Update the UI to reflect the new room
			rw.UI.refreshNetworkList()
		} else {
			// If room ID is empty for some reason, show generic success
			dialog.ShowInformation("Network Created", "Network created successfully!", rw.Window)
		}

		// Close the window after successful creation
		rw.Close()

		// Define a referência no UIManager como nil para garantir que uma nova instância seja criada na próxima vez
		rw.UI.RoomWindow = nil
	}()
}

// Close sobrescreve o método Close da BaseWindow para garantir que a referência no UIManager seja limpa
func (rw *RoomWindow) Close() {
	// Chama o método Close da classe pai
	rw.BaseWindow.Close()

	// Limpa a referência no UIManager
	rw.UI.RoomWindow = nil
}
