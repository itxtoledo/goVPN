package main

import (
	"log"
	"sort"
	"sync"

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
	updateMutex   sync.Mutex
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
	// Use mutex to prevent concurrent modifications
	ntc.updateMutex.Lock()
	defer ntc.updateMutex.Unlock()

	// Limpar o accordion existente
	ntc.RoomAccordion.CloseAll()

	// Criar estrutura temporária para os novos itens
	var newItems []*widget.AccordionItem

	// Ordenar as salas por nome
	if ntc.UI.Rooms != nil {
		sort.Slice(ntc.UI.Rooms, func(i, j int) bool {
			return ntc.UI.Rooms[i].Name < ntc.UI.Rooms[j].Name
		})
	}

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

	// Imprimir log para debug
	log.Printf("Atualizando lista de salas. Total: %d", len(ntc.UI.Rooms))

	// Adicionar cada sala como um item de accordion
	if ntc.UI.Rooms != nil && len(ntc.UI.Rooms) > 0 {
		for _, room := range ntc.UI.Rooms {
			log.Printf("Processando sala: %s (ID=%s)", room.Name, room.ID)
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
			// Change button label based on whether the user is already connected to the room
			connectButtonText := "Connect"
			connectIcon := theme.LoginIcon()

			// If already connected to this room, use "Disconnect" text and different icon
			if isConnected {
				connectButtonText = "Disconnect"
				connectIcon = theme.LogoutIcon()
			}

			connectButton := widget.NewButtonWithIcon(connectButtonText, connectIcon, func(currentRoom *storage.Room) func() {
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
								dialog.ShowInformation("Success", "Successfully left room: "+currentRoom.Name, ntc.UI.MainWindow)
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
				// Show a different indicator for rooms the user is a member of but not currently connected to
				titleText = "🔵 " + room.Name // Blue circle for joined but not connected
			}

			// Criar item de accordion para a sala
			accordionItem := widget.NewAccordionItem(titleText, content)
			newItems = append(newItems, accordionItem)

			// Track which items should be open
			if isConnected {
				accordionItem.Open = true
			}
		}
	} else {
		log.Printf("Nenhuma sala disponível para exibir")
		// Adicionar mensagem informativa quando não há salas
		accordionItem := widget.NewAccordionItem("No Rooms Available", widget.NewLabel("Create or join a room to get started."))
		newItems = append(newItems, accordionItem)
		accordionItem.Open = true
	}

	// Atualizar o accordion com os novos itens
	ntc.RoomAccordion.Items = newItems

	// Atualizar a apresentação
	ntc.RoomAccordion.Refresh()
	ntc.Container.Refresh()
}

// GetContainer retorna o container principal
func (ntc *NetworkListComponent) GetContainer() *fyne.Container {
	return ntc.Container
}
