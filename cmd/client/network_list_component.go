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
		nil, // widget.NewLabelWithStyle("Available Rooms", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: false}),
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

	if len(rooms) > 0 {
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

			// Create connected computers list
			computersContainer := container.NewVBox()

			// Add current computer to the list
			currentComputerItem := container.NewHBox(
				widget.NewLabel("🟢"),
				widget.NewLabel(username+" (you)"),
			)

			computersContainer.Add(currentComputerItem)

			// Add other connected computers if we're connected to this room
			if isConnected && ntc.UI.VPN.NetworkManager != nil {
				// Get computers in the room from the NetworkManager
				for _, computer := range ntc.UI.VPN.NetworkManager.Computers {
					// Skip our own computer
					if computer.OwnerID == ntc.UI.VPN.PublicKeyStr {
						continue
					}

					// Create status icon based on online status
					var statusIcon string
					if computer.IsOnline {
						statusIcon = "🟢"
					} else {
						statusIcon = "🔴"
					}

					// Create computer item with icon, status and name
					computerItem := container.NewHBox(
						widget.NewIcon(theme.AccountIcon()),
						widget.NewLabel(statusIcon),
						widget.NewLabel(computer.Name),
					)
					computersContainer.Add(computerItem)
				}
			}

			// Create computers section
			computersBox := computersContainer

			// Create buttons for room actions with improved styling
			connectButtonText := "🔗 Connect"
			connectIcon := theme.LoginIcon()

			// If already connected to this room, use "Disconnect" text and different icon
			if isConnected {
				connectButtonText = "🔌 Disconnect"
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

			leaveButton := widget.NewButtonWithIcon("🚪 Leave", theme.LogoutIcon(), func(currentRoom *storage.Room) func() {
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

			// Style buttons based on connection status
			if isConnected {
				connectButton.Importance = widget.HighImportance
			} else {
				connectButton.Importance = widget.MediumImportance
			}

			// Create actions section
			actionsBox := container.NewVBox(
				widget.NewLabelWithStyle("Actions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewSeparator(),
				container.NewHBox(
					layout.NewSpacer(),
					connectButton,
					leaveButton,
				),
			)

			// Create the main content with clear visual separation and horizontal padding
			content := container.NewPadded(
				container.NewVBox(
					computersBox,
					widget.NewSeparator(),
					actionsBox,
				),
			)

			// Create title with just icon and name
			var titleIcon string
			if isConnected {
				titleIcon = "🟢"
			} else {
				titleIcon = "🔵"
			}

			// Create title with icon and room name only
			fullTitle := fmt.Sprintf("%s %s", titleIcon, room.Name)

			// Create accordion item for the room
			accordionItem := widget.NewAccordionItem(fullTitle, content)
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
			log.Printf("No rooms available to display after filtering/processing") // Add informative message when no rooms are available
			noRoomsLabel := widget.NewLabelWithStyle(
				"No rooms available.\nCreate or join a room to get started.",
				fyne.TextAlignCenter,
				fyne.TextStyle{Italic: true},
			)
			ntc.contentContainer.Add(container.NewCenter(noRoomsLabel)) // Add centered label
		}
	} else {
		log.Printf("No rooms available to display")
		// Add informative message when no rooms are available
		noRoomsLabel := widget.NewLabelWithStyle(
			"No rooms available.\nCreate or join a room to get started.",
			fyne.TextAlignCenter,
			fyne.TextStyle{Italic: true},
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

// CustomRoomTitle creates a clickable room title with context menu
type CustomRoomTitle struct {
	widget.BaseWidget
	Text        string
	Room        *storage.Room
	IsConnected bool
	UI          *UIManager
}

func NewCustomRoomTitle(text string, room *storage.Room, isConnected bool, ui *UIManager) *CustomRoomTitle {
	title := &CustomRoomTitle{
		Text:        text,
		Room:        room,
		IsConnected: isConnected,
		UI:          ui,
	}
	title.ExtendBaseWidget(title)
	return title
}

func (c *CustomRoomTitle) CreateRenderer() fyne.WidgetRenderer {
	text := widget.NewLabel(c.Text)
	text.TextStyle = fyne.TextStyle{Bold: true}

	return widget.NewSimpleRenderer(text)
}

func (c *CustomRoomTitle) Tapped(pe *fyne.PointEvent) {
	// Handle normal click - expand/collapse accordion
}

func (c *CustomRoomTitle) TappedSecondary(pe *fyne.PointEvent) {
	// Handle right-click - show context menu
	menu := fyne.NewMenu("Room Actions")

	// Add connect/disconnect option
	if c.IsConnected {
		disconnectItem := fyne.NewMenuItem("🔌 Disconnect", func() {
			c.UI.SelectedRoom = c.Room
			log.Println("Disconnecting from room:", c.Room.Name)
			go func() {
				err := c.UI.VPN.NetworkManager.DisconnectRoom(c.Room.ID)
				if err != nil {
					log.Printf("Error disconnecting from room: %v", err)
					dialog.ShowError(fmt.Errorf("failed to disconnect from room: %v", err), c.UI.MainWindow)
				} else {
					log.Println("Successfully disconnected from room.")
					dialog.ShowInformation("Success", "Successfully disconnected from room.", c.UI.MainWindow)
				}
			}()
		})
		menu.Items = append(menu.Items, disconnectItem)
	} else {
		connectItem := fyne.NewMenuItem("🔗 Connect", func() {
			c.UI.SelectedRoom = c.Room
			if c.UI.ConnectDialog == nil {
				c.UI.ConnectDialog = dialogs.NewConnectDialog(c.UI, c.UI.VPN.Username)
			}
			c.UI.ConnectDialog.Show()
		})
		menu.Items = append(menu.Items, connectItem)
	}

	// Add leave option
	leaveItem := fyne.NewMenuItem("🚪 Leave", func() {
		if c.UI.VPN.NetworkManager != nil {
			go func() {
				err := c.UI.VPN.NetworkManager.LeaveRoomById(c.Room.ID)
				if err != nil {
					log.Printf("Error deleting room: %v", err)
					fyne.CurrentApp().SendNotification(&fyne.Notification{
						Title:   "Error",
						Content: "Failed to leave room: " + err.Error(),
					})
				} else {
					log.Println("Successfully left room:", c.Room.Name)
					fyne.CurrentApp().SendNotification(&fyne.Notification{
						Title:   "Success",
						Content: "Successfully left room: " + c.Room.Name,
					})
					dialog.ShowInformation("Success", "Successfully left room: "+c.Room.Name, c.UI.MainWindow)
				}
			}()
		}
	})
	menu.Items = append(menu.Items, leaveItem)

	// Show context menu
	widget.ShowPopUpMenuAtPosition(menu, c.UI.MainWindow.Canvas(), pe.AbsolutePosition)
}
