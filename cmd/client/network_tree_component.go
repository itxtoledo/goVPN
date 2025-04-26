// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/network_tree_component.go
package main

import (
	"log"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// NetworkTreeComponent representa o componente da árvore de rede
type NetworkTreeComponent struct {
	UI        *UIManager
	Container *fyne.Container
	RoomList  *widget.List
}

// NewNetworkTreeComponent cria uma nova instância do componente de árvore de rede
func NewNetworkTreeComponent(ui *UIManager) *NetworkTreeComponent {
	ntc := &NetworkTreeComponent{
		UI: ui,
	}
	ntc.init()
	return ntc
}

// init inicializa o componente
func (ntc *NetworkTreeComponent) init() {
	// Criar lista de salas
	ntc.RoomList = widget.NewList(
		func() int {
			if ntc.UI.Rooms == nil {
				return 0
			}
			return len(ntc.UI.Rooms)
		},
		func() fyne.CanvasObject {
			// Criar um novo item de sala para a lista
			if ntc.UI.RoomItemComponent == nil {
				ntc.UI.RoomItemComponent = NewRoomItemComponent(ntc.UI)
			}
			return ntc.UI.RoomItemComponent.CreateRoomItem()
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			// Atualizar o item de sala com os dados da sala
			if id < 0 || id >= len(ntc.UI.Rooms) {
				return
			}

			// Obter a sala pelo ID
			room := ntc.UI.Rooms[id]

			// Converter o objeto genérico para o tipo específico
			roomItem, ok := item.(*fyne.Container)
			if !ok {
				log.Printf("Error: item is not a Container")
				return
			}

			// Atualizar os labels dentro do container
			for _, obj := range roomItem.Objects {
				container, ok := obj.(*fyne.Container)
				if !ok {
					continue
				}

				// Procurar labels dentro do container
				if textContainer, ok := container.Objects[0].(*fyne.Container); ok {
					// Primeiro label é o nome da sala
					if nameLabel, ok := textContainer.Objects[0].(*widget.Label); ok {
						nameLabel.Text = room.Name
						nameLabel.Refresh()
					}

					// Segundo label é a descrição da sala
					if len(textContainer.Objects) > 1 {
						if descLabel, ok := textContainer.Objects[1].(*widget.Label); ok {
							descLabel.Text = room.Description
							descLabel.Refresh()
						}
					}
				}
			}
		},
	)

	// Configurar o callback de seleção
	ntc.RoomList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(ntc.UI.Rooms) {
			// Armazenar a sala selecionada
			ntc.UI.SelectedRoom = ntc.UI.Rooms[id]

			// Mostrar diálogo de conexão
			if ntc.UI.ConnectDialog == nil {
				ntc.UI.ConnectDialog = NewConnectDialog(ntc.UI)
			}
			ntc.UI.ConnectDialog.Show()

			// Desselecionar após o clique
			ntc.RoomList.UnselectAll()
		}
	}

	// Criar o container principal
	ntc.Container = container.NewBorder(
		widget.NewLabelWithStyle("Available Rooms", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil,
		nil,
		nil,
		ntc.RoomList,
	)
}

// updateNetworkList atualiza a lista de redes
func (ntc *NetworkTreeComponent) updateNetworkList() {
	// Ordenar as salas por nome
	if ntc.UI.Rooms != nil {
		sort.Slice(ntc.UI.Rooms, func(i, j int) bool {
			return ntc.UI.Rooms[i].Name < ntc.UI.Rooms[j].Name
		})
	}

	// Atualizar a lista
	ntc.RoomList.Refresh()
}

// GetContainer retorna o container principal
func (ntc *NetworkTreeComponent) GetContainer() *fyne.Container {
	return ntc.Container
}
