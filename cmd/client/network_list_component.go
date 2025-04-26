package main

import (
	"log"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// NetworkListComponent representa o componente da árvore de rede
type NetworkListComponent struct {
	UI            *UIManager
	Container     *fyne.Container
	RoomAccordion *widget.Accordion
}

// NewNetworkListComponent cria uma nova instância do componente de árvore de rede
func NewNetworkListComponent(ui *UIManager) *NetworkListComponent {
	ntc := &NetworkListComponent{
		UI: ui,
	}
	ntc.init()
	return ntc
}

// init inicializa o componente
func (ntc *NetworkListComponent) init() {
	// Criar um accordion vazio
	ntc.RoomAccordion = widget.NewAccordion()

	// Criar o container principal
	ntc.Container = container.NewBorder(
		widget.NewLabelWithStyle("Available Rooms", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil,
		nil,
		nil,
		ntc.RoomAccordion,
	)
}

// updateNetworkList atualiza a lista de redes
func (ntc *NetworkListComponent) updateNetworkList() {
	// Ordenar as salas por nome
	if ntc.UI.Rooms != nil {
		sort.Slice(ntc.UI.Rooms, func(i, j int) bool {
			return ntc.UI.Rooms[i].Name < ntc.UI.Rooms[j].Name
		})
	}

	// Limpar o accordion
	ntc.RoomAccordion.CloseAll()
	ntc.RoomAccordion.Items = []*widget.AccordionItem{}

	// Adicionar cada sala como um item de accordion
	if ntc.UI.Rooms != nil {
		for _, room := range ntc.UI.Rooms {
			// Criar botões para as ações da sala
			connectButton := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func(currentRoom *storage.Room) func() {
				return func() {
					ntc.UI.SelectedRoom = currentRoom

					// Mostrar diálogo de conexão
					if ntc.UI.ConnectDialog == nil {
						ntc.UI.ConnectDialog = NewConnectDialog(ntc.UI)
					}
					ntc.UI.ConnectDialog.Show()
				}
			}(room))

			leaveButton := widget.NewButtonWithIcon("Leave", theme.LogoutIcon(), func(currentRoom *storage.Room) func() {
				return func() {
					// Delegate deletion to NetworkManager
					if ntc.UI.VPN.NetworkManager != nil {
						go func() {
							err := ntc.UI.VPN.NetworkManager.LeaveRoomById(currentRoom.ID)
							if err != nil {
								log.Printf("Error deleting room: %v", err)
								fyne.CurrentApp().SendNotification(&fyne.Notification{
									Title:   "Error",
									Content: "Failed to leave room: " + err.Error(),
								})
							} else {
								log.Println("Successfully left room:", currentRoom.Name)
								fyne.CurrentApp().SendNotification(&fyne.Notification{
									Title:   "Success",
									Content: "Successfully left room: " + currentRoom.Name,
								})

								// Show success dialog on the main thread
								dialog.NewInformation("Success", "Successfully left room: "+currentRoom.Name, ntc.UI.MainWindow).Show()
							}
						}()
					}
				}
			}(room))

			// Criar layout para os botões
			buttonBox := container.NewHBox(
				layout.NewSpacer(),
				connectButton,
				leaveButton,
			)

			content := container.NewVBox(
				widget.NewLabel("Last connected: "+room.LastConnected.Local().String()),
				buttonBox,
			)

			// Criar item de accordion para a sala
			accordionItem := widget.NewAccordionItem(room.Name, content)
			ntc.RoomAccordion.Append(accordionItem)
		}
	}

	// Atualizar o accordion
	ntc.RoomAccordion.Refresh()
}

// GetContainer retorna o container principal
func (ntc *NetworkListComponent) GetContainer() *fyne.Container {
	return ntc.Container
}
