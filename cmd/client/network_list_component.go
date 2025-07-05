package main

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/dialogs"
	"github.com/itxtoledo/govpn/cmd/client/storage"
)

// NetworkListComponent representa o componente da árvore de rede
type NetworkListComponent struct {
	UI               *UIManager
	Container        *fyne.Container
	RoomAccordion    *widget.Accordion
	contentContainer *fyne.Container // New field to hold dynamic content
	updateMutex      sync.Mutex
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
	// Initialize the dynamic content container
	ntc.contentContainer = container.NewStack()

	// Criar o container principal
	ntc.Container = container.NewBorder(
		widget.NewLabelWithStyle("Available Rooms", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil,
		nil,
		nil,
		ntc.contentContainer, // Use the new contentContainer here
	)
}

// updateNetworkList atualiza a lista de redes
func (ntc *NetworkListComponent) updateNetworkList() {
	// Use mutex to prevent concurrent modifications
	ntc.updateMutex.Lock()
	defer ntc.updateMutex.Unlock()

	// Clear the content container before adding new content
	ntc.contentContainer.RemoveAll()

	// Sort the rooms by name
	rooms := ntc.UI.RealtimeData.GetRooms()
	if rooms != nil {
		sort.Slice(rooms, func(i, j int) bool {
			return rooms[i].Name < rooms[j].Name
		})
	}

	log.Printf("Updating room list. Total: %d", len(rooms))

	if rooms != nil && len(rooms) > 0 {
		// Criar estrutura temporária para os novos itens
		var newItems []*widget.AccordionItem

		// Get the current room ID from the network manager (if connected)
		currentRoomID := ""
		if ntc.UI.VPN.NetworkManager != nil {
			currentRoomID = ntc.UI.VPN.NetworkManager.RoomID
		}

		// Get username from config for display
		username, _ := ntc.UI.RealtimeData.Username.Get()
		if username == "" {
			username = "You"
		}

		// Add each room as an accordion item
		for _, room := range rooms {
			log.Printf("Processing room: %s (ID=%s)", room.Name, room.ID)
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

			// Create buttons for room actions
			// Change button label based on whether the user is already connected to the room
			connectButtonText := "Connect"
			connectIcon := theme.LoginIcon()

			// If already connected to this room, use "Disconnect" text and different icon
			if isConnected {
				connectButtonText = "Disconnect"
				connectIcon = theme.LogoutIcon()
			}

			connectButton := widget.NewButtonWithIcon(connectButtonText, connectIcon, func(currentRoom *storage.Room, isConnected bool) func() {
				return func() {
					ntc.UI.SelectedRoom = currentRoom

					if isConnected {
						// If already connected, disconnect
						log.Println("Disconnecting from room:", currentRoom.Name)
						go func() {
							err := ntc.UI.VPN.NetworkManager.DisconnectRoom(currentRoom.ID)
							if err != nil {
								log.Printf("Error disconnecting from room: %v", err)
								dialog.ShowError(fmt.Errorf("failed to disconnect from room: %v", err), ntc.UI.MainWindow)
							} else {
								log.Println("Successfully disconnected from room.")
								dialog.ShowInformation("Success", "Successfully disconnected from room.", ntc.UI.MainWindow)
							}
						}()
					} else {
						// Show connection dialog
						if ntc.UI.ConnectDialog == nil {
							ntc.UI.ConnectDialog = dialogs.NewConnectDialog(ntc.UI, ntc.UI.VPN.Username)
						}
						ntc.UI.ConnectDialog.Show()
					}
				}
			}(room, isConnected))

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

			// Create layout for buttons
			buttonBox := container.NewHBox(
				layout.NewSpacer(),
				connectButton,
				leaveButton,
			)

			// Create room info container
			infoBox := container.NewVBox(
				membersBox,
				widget.NewLabel("Last connected: "+room.LastConnected.Format("2006-01-02 15:04:05")),
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

			// Create accordion item for the room
			accordionItem := widget.NewAccordionItem(titleText, content)
			newItems = append(newItems, accordionItem)

			// Track which items should be open
			if isConnected {
				accordionItem.Open = true
			}
		}
		// Update the accordion with the new items
		ntc.RoomAccordion.Items = newItems

		if len(newItems) > 0 {
			// Add the accordion to the content container
			ntc.contentContainer.Add(ntc.RoomAccordion)
			// Refresh the accordion presentation
			ntc.RoomAccordion.Refresh()
		} else {
			log.Printf("No rooms available to display after filtering/processing")
			// Add informative message when no rooms are available
			noRoomsLabel := widget.NewLabelWithStyle(
				"No rooms available.\nCreate or join a room to get started.",
				fyne.TextAlignCenter,
				fyne.TextStyle{},
			)
			ntc.contentContainer.Add(container.NewCenter(noRoomsLabel)) // Add centered label
		}
	} else {
		log.Printf("No rooms available to display")
		// Add informative message when no rooms are available
		noRoomsLabel := widget.NewLabelWithStyle(
			"No rooms available.\nCreate or join a room to get started.",
			fyne.TextAlignCenter,
			fyne.TextStyle{},
		)
		ntc.contentContainer.Add(container.NewCenter(noRoomsLabel)) // Add centered label
	}

	// Refresh the content container and main container
	ntc.contentContainer.Refresh()
	ntc.Container.Refresh()
}

// GetContainer retorna o container principal
func (ntc *NetworkListComponent) GetContainer() *fyne.Container {
	return ntc.Container
}
