// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/home_tab_component.go
package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// HomeTabComponent representa o componente da aba principal
type HomeTabComponent struct {
	UI *UIManager
}

// NewHomeTabComponent cria uma nova instância do componente da aba principal
func NewHomeTabComponent(ui *UIManager) *HomeTabComponent {
	htc := &HomeTabComponent{
		UI: ui,
	}
	return htc
}

// CreateHomeTabContainer cria o container principal da aba inicial
func (htc *HomeTabComponent) CreateHomeTabContainer() *fyne.Container {
	// Criar o container de salas disponíveis
	roomsContainer := htc.UI.NetworkTreeComp.GetContainer()

	// Criar um botão para criar uma nova sala
	createRoomButton := widget.NewButtonWithIcon("Create Room", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "contentAdd"), func() {
		log.Println("Create room button clicked")

		// Check network connection status
		if htc.UI.VPN != nil && htc.UI.VPN.NetworkManager != nil {
			isConnected := htc.UI.VPN.NetworkManager.GetConnectionState() == ConnectionStateConnected
			if !isConnected {
				dialog.ShowError(fmt.Errorf("not connected to server"), htc.UI.MainWindow)
				return
			}
		} else {
			dialog.ShowError(fmt.Errorf("network manager not initialized"), htc.UI.MainWindow)
			return
		}

		// Create and show the room creation window
		createRoomWindow := NewCreateRoomDialog(htc.UI)
		createRoomWindow.Show()
	})

	// Criar um botão para entrar em uma sala
	joinRoomButton := widget.NewButtonWithIcon("Join Room", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "mail-reply"), func() {
		log.Println("Join room button clicked")

		// Check network connection status
		if htc.UI.VPN != nil && htc.UI.VPN.NetworkManager != nil {
			isConnected := htc.UI.VPN.NetworkManager.GetConnectionState() == ConnectionStateConnected
			if !isConnected {
				dialog.ShowError(fmt.Errorf("not connected to server"), htc.UI.MainWindow)
				return
			}
		} else {
			dialog.ShowError(fmt.Errorf("network manager not initialized"), htc.UI.MainWindow)
			return
		}

		// Create and show the room joining window
		joinRoomWindow := NewJoinRoomDialog(htc.UI)
		joinRoomWindow.Show()
	})

	// Criar o container da aba principal
	mainContainer := container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), joinRoomButton, createRoomButton),
		nil,
		nil,
		roomsContainer,
	)

	return mainContainer
}
