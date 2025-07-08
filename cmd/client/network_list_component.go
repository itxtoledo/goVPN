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
	RoomAccordion    *CustomAccordion
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
	// Criar um accordion personalizado vazio
	ntc.RoomAccordion = NewCustomAccordion()
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

// UpdateNetworkList atualiza a lista de redes
func (ntc *NetworkListComponent) UpdateNetworkList() {
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
		// Clear the accordion before adding new items
		ntc.RoomAccordion.RemoveAll()

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

			// Add current computer to the list (always show if we're in this room)
			var currentStatusIndicator fyne.CanvasObject
			if isConnected {
				currentActivity := widget.NewActivity()
				currentActivity.Start() // Green and animated when connected/online
				currentStatusIndicator = currentActivity
			} else {
				// Show RadioButtonFillIcon when not connected
				currentStatusIndicator = widget.NewIcon(theme.RadioButtonFillIcon())
			}

			currentComputerItem := container.NewHBox(
				currentStatusIndicator,
				widget.NewLabel(username+" (you)"),
			)
			computersContainer.Add(currentComputerItem)

			// Add other computers in the room
			// Add other computers in the room
			if ntc.UI.VPN.NetworkManager != nil && len(ntc.UI.VPN.NetworkManager.Computers) > 0 {
				// Get computers in the room from the NetworkManager
				for _, computer := range ntc.UI.VPN.NetworkManager.Computers {
					// Skip our own computer
					if computer.OwnerID == ntc.UI.VPN.PublicKeyStr {
						continue
					}

					// Create activity indicator based on online status
					var activity fyne.CanvasObject
					if computer.IsOnline {
						activityWidget := widget.NewActivity()
						activityWidget.Start() // Green and animated when online
						activity = activityWidget
					} else {
						// Show RadioButtonFillIcon when offline
						activity = widget.NewIcon(theme.RadioButtonFillIcon())
					}

					// Create computer item with icon, activity indicator and name
					computerItem := container.NewHBox(
						widget.NewIcon(theme.AccountIcon()),
						activity,
						widget.NewLabel(computer.Name),
					)
					computersContainer.Add(computerItem)
				}
			}

			// Create computers section
			computersBox := computersContainer

			// Create buttons for room actions with improved styling
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

			// Style buttons based on connection status
			if isConnected {
				connectButton.Importance = widget.HighImportance
			} else {
				connectButton.Importance = widget.MediumImportance
			}

			// Create actions section
			actionsBox := container.NewVBox(
				container.NewHBox(
					layout.NewSpacer(),
					connectButton,
					leaveButton,
				),
			)

			content := container.NewPadded(
				container.NewHBox(
					layout.NewSpacer(),
					container.NewVBox(
						computersBox,
						widget.NewSeparator(),
						actionsBox,
					),
					layout.NewSpacer(),
				),
			)

			// Create custom title with activity indicator or radio button icon
			var statusIndicator fyne.CanvasObject
			if isConnected {
				titleActivity := widget.NewActivity()
				titleActivity.Start()
				statusIndicator = titleActivity
			} else {
				// Show RadioButtonFillIcon when not connected
				statusIndicator = widget.NewIcon(theme.RadioButtonFillIcon())
			}

			// Calculate connected users count
			connectedUsers := 0
			if isConnected {
				connectedUsers = 1 // Count yourself if connected
			}

			// Count other computers that are online
			if ntc.UI.VPN.NetworkManager != nil && len(ntc.UI.VPN.NetworkManager.Computers) > 0 {
				for _, computer := range ntc.UI.VPN.NetworkManager.Computers {
					if computer.OwnerID != ntc.UI.VPN.PublicKeyStr && computer.IsOnline {
						connectedUsers++
					}
				}
			}

			titleLabel := widget.NewLabelWithStyle(room.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			userCountLabel := widget.NewLabelWithStyle(fmt.Sprintf("(%d/10)", connectedUsers), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

			customTitle := container.NewHBox(
				statusIndicator,
				titleLabel,
			)

			// Create custom accordion item with context menu support and user count
			accordionItem := NewCustomAccordionItemWithEndContentAndCallbacks(customTitle, content, userCountLabel, nil)

			// Auto-open if connected
			if isConnected {
				accordionItem.Open()
			}

			// Add item to accordion
			ntc.RoomAccordion.AddItem(accordionItem)
		}

		if len(ntc.RoomAccordion.Items) > 0 {
			// Add the accordion container to the content container
			ntc.contentContainer.Add(ntc.RoomAccordion.GetContainer())
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

// CustomRoomItem creates a custom expandable room item with activity indicator in title
type CustomRoomItem struct {
	widget.BaseWidget
	Room        *storage.Room
	IsConnected bool
	UI          *UIManager
	Content     *fyne.Container
	Activity    *widget.Activity
	IsExpanded  bool
	titleLabel  *widget.Label
	expandBtn   *widget.Button
	container   *fyne.Container
}

func NewCustomRoomItem(room *storage.Room, isConnected bool, ui *UIManager, content *fyne.Container) *CustomRoomItem {
	item := &CustomRoomItem{
		Room:        room,
		IsConnected: isConnected,
		UI:          ui,
		Content:     content,
		IsExpanded:  isConnected, // Auto-expand if connected
	}

	item.Activity = widget.NewActivity()
	item.Activity.Start()
	if isConnected {
		item.Activity.Start()
	} else {
		item.Activity.Stop()
	}

	item.titleLabel = widget.NewLabel(room.Name)
	item.titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	item.expandBtn = widget.NewButton("▼", func() {
		item.Toggle()
	})
	item.expandBtn.Importance = widget.LowImportance

	item.ExtendBaseWidget(item)
	item.updateUI()
	return item
}

func (c *CustomRoomItem) Toggle() {
	c.IsExpanded = !c.IsExpanded
	c.updateUI()
	c.Refresh()
}

func (c *CustomRoomItem) updateUI() {
	// Update expand button text
	if c.IsExpanded {
		c.expandBtn.SetText("▲")
	} else {
		c.expandBtn.SetText("▼")
	}

	// Create title row with activity indicator
	titleRow := container.NewHBox(
		c.expandBtn,
		c.Activity,
		c.titleLabel,
		layout.NewSpacer(),
	)

	if c.IsExpanded {
		c.container = container.NewVBox(
			titleRow,
			c.Content,
		)
	} else {
		c.container = container.NewVBox(titleRow)
	}
}

func (c *CustomRoomItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.container)
}

func (c *CustomRoomItem) UpdateConnectionStatus(isConnected bool) {
	c.IsConnected = isConnected
	if isConnected {
		c.Activity.Start()
	} else {
		c.Activity.Stop()
	}
	c.Refresh()
}
