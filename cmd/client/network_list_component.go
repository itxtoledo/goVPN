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

	// Get the current room ID from the network manager (if connected)
	currentRoomID := ""
	if ntc.UI.VPN.NetworkManager != nil {
		currentRoomID = ntc.UI.VPN.NetworkManager.RoomID
	}

	// Get username from config for display
	username := ntc.UI.ConfigManager.GetConfig().Username
	if username == "" {
		username = "You"
	}

	// Adicionar cada sala como um item de accordion
	if ntc.UI.Rooms != nil {
		for _, room := range ntc.UI.Rooms {
			// Check if this room is the one we're currently connected to
			isConnected := room.ID == currentRoomID

			// Create members container
			membersContainer := container.NewVBox()

			// Add current user to members list
			userLabel := widget.NewLabel("• " + username + " (you)")
			membersContainer.Add(userLabel)

			// Add other room members if we're connected to this room
			if isConnected && ntc.UI.VPN.NetworkManager != nil {
				// Get computers in the room from the NetworkManager
				for _, computer := range ntc.UI.VPN.NetworkManager.Computers {
					// Skip our own computer
					if computer.OwnerID == ntc.UI.VPN.PublicKeyStr {
						continue
					}

					// Create status text based on online status
					statusText := ""
					if computer.IsOnline {
						statusText = " (online)"
					} else {
						statusText = " (offline)"
					}

					// Add this computer to the members list
					computerLabel := widget.NewLabel("• " + computer.Name + statusText)
					membersContainer.Add(computerLabel)
				}
			}

			// Add "Members" header
			membersBox := container.NewVBox(
				widget.NewLabelWithStyle("Room Members:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				membersContainer,
			)

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

			// Disable connect button if already connected to this room
			if isConnected {
				connectButton.Disable()
			}

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

			// Create room info container
			infoBox := container.NewVBox(
				membersBox,
				widget.NewLabel("Last connected: "+room.LastConnected.Local().String()),
			)

			content := container.NewVBox(
				infoBox,
				buttonBox,
			)

			// Create title with status indicator
			var titleText string
			if isConnected {
				titleText = "🟢 " + room.Name // Green circle for connected
			} else {
				titleText = "⚪ " + room.Name // White circle for available
			}

			// Criar item de accordion para a sala
			accordionItem := widget.NewAccordionItem(titleText, content)
			ntc.RoomAccordion.Append(accordionItem)

			// Open the accordion for the currently connected room
			if isConnected {
				accordionItem.Open = true
			}
		}
	}

	// Atualizar o accordion
	ntc.RoomAccordion.Refresh()
}

// GetContainer retorna o container principal
func (ntc *NetworkListComponent) GetContainer() *fyne.Container {
	return ntc.Container
}
