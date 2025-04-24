// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/network_tab_component.go
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// NetworkTabComponent representa o conteúdo da aba Network
type NetworkTabComponent struct {
	UI            *UIManager
	Container     *fyne.Container
	RefreshButton *widget.Button
	RoomsButton   *widget.Button
}

// NewNetworkTabComponent cria uma nova instância do componente da aba Network
func NewNetworkTabComponent(ui *UIManager) *NetworkTabComponent {
	comp := &NetworkTabComponent{
		UI: ui,
	}

	comp.createContent()

	return comp
}

// createContent cria o conteúdo da aba Network
func (n *NetworkTabComponent) createContent() {
	// Botão para atualizar lista
	n.RefreshButton = widget.NewButton("Atualizar Lista", func() {
		n.UI.refreshNetworkList()
	})

	// Botão para gerenciar salas
	n.RoomsButton = widget.NewButton("Gerenciar Salas", func() {
		if n.UI.RoomWindow == nil {
			n.UI.RoomWindow = NewRoomWindow(n.UI)
		}
		n.UI.RoomWindow.Show()
	})

	// Container para os botões
	networkButtons := container.NewHBox(
		n.RefreshButton,
		layout.NewSpacer(),
		n.RoomsButton,
	)

	// Criando o conteúdo da aba
	n.Container = container.NewVBox(
		widget.NewLabelWithStyle("Rede", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		networkButtons,
		widget.NewSeparator(),
		widget.NewLabel("Lista de usuários conectados:"),
		container.NewVBox(widget.NewLabel("Sem usuários conectados")), // Placeholder para lista de usuários
	)
}

// GetContainer retorna o container da aba Network
func (n *NetworkTabComponent) GetContainer() *fyne.Container {
	return n.Container
}
