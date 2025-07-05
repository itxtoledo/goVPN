// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/room_item_component.go
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// RoomItemComponent representa o componente de item de sala na lista de salas
type RoomItemComponent struct {
}

// NewRoomItemComponent cria uma nova instância do componente de item de sala
func NewRoomItemComponent(ui *UIManager) *RoomItemComponent {
	return &RoomItemComponent{}
}

// CreateRoomItem cria um novo item de sala para a lista
func (ric *RoomItemComponent) CreateRoomItem() fyne.CanvasObject {
	// Criar os labels para o nome e descrição da sala
	roomNameLabel := widget.NewLabel("")
	roomNameLabel.TextStyle = fyne.TextStyle{Bold: true}

	roomDescLabel := widget.NewLabel("")
	roomDescLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Criar o container para os labels de texto
	textContainer := container.NewVBox(
		roomNameLabel,
		roomDescLabel,
	)

	// Ícone para indicar se a sala tem senha ou não
	lockIcon := widget.NewIcon(theme.VisibilityIcon())

	// Container para o ícone
	iconContainer := container.New(layout.NewCenterLayout(), lockIcon)

	// Container principal com layout horizontal
	mainContainer := container.NewBorder(
		nil, nil, nil, iconContainer,
		textContainer,
	)

	// Adicionar um padding ao redor do item
	paddedContainer := container.NewPadded(mainContainer)

	return paddedContainer
}
