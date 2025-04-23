package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// RoomWindow manages the VPN rooms interface
type RoomWindow struct {
	UI                *UIManager
	Window            fyne.Window
	RoomList          *widget.List
	Rooms             []models.Room
	CreateBtn         *widget.Button
	JoinBtn           *widget.Button
	RefreshBtn        *widget.Button
	Container         *fyne.Container
	selectedRoomIndex int // Stores the index of the selected room
}

// NewRoomWindow creates a new rooms window
func NewRoomWindow(ui *UIManager) *RoomWindow {
	roomWindow := &RoomWindow{
		UI:                ui,
		Rooms:             []models.Room{},
		Window:            ui.createWindow("Rooms - goVPN", 600, 400, false),
		selectedRoomIndex: -1, // No room selected initially
	}

	// Sets up the room list update
	ui.VPN.NetworkManager.OnRoomListUpdate = roomWindow.updateRoomList

	return roomWindow
}

// Show displays the rooms window
func (rw *RoomWindow) Show() {
	// If the window has been closed, recreate it
	if rw.Window == nil {
		rw.Window = rw.UI.createWindow("Rooms - goVPN", 600, 400, false)
	}

	// Ensures that the window content is updated
	if rw.Window.Content() == nil {
		rw.Window.SetContent(rw.CreateContent())
	}

	// Updates the room list
	rw.refreshRoomList()

	rw.Window.Show()
	rw.Window.CenterOnScreen()
}

// CreateContent creates the content for the rooms window
func (rw *RoomWindow) CreateContent() fyne.CanvasObject {
	// Room list
	rw.RoomList = widget.NewList(
		func() int { return len(rw.Rooms) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Room Name"),
				widget.NewLabel("Users"),
				widget.NewLabel("Status"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			room := rw.Rooms[id]
			labels := item.(*fyne.Container).Objects
			statusText := "Available"
			if room.IsCreator {
				statusText = "Created by you"
			}

			labels[0].(*widget.Label).SetText(room.Name)
			labels[1].(*widget.Label).SetText(fmt.Sprintf("%d", room.ClientCount))
			labels[2].(*widget.Label).SetText(statusText)
		},
	)

	// Adds selection to the room list
	rw.RoomList.OnSelected = func(id widget.ListItemID) {
		// Stores the selected index
		rw.selectedRoomIndex = id
		// Enables the join button when a room is selected
		rw.JoinBtn.Enable()
	}

	// Adds a handler to remove selection
	rw.RoomList.OnUnselected = func(id widget.ListItemID) {
		rw.selectedRoomIndex = -1
		rw.JoinBtn.Disable()
	}

	// Action buttons
	rw.CreateBtn = widget.NewButton("Create Room", rw.showCreateRoomDialog)
	rw.JoinBtn = widget.NewButton("Join", rw.showJoinRoomDialog)
	rw.JoinBtn.Disable() // Initially disabled until a room is selected

	rw.RefreshBtn = widget.NewButton("Refresh List", rw.refreshRoomList)

	buttonBar := container.NewHBox(
		rw.CreateBtn,
		rw.JoinBtn,
		rw.RefreshBtn,
	)

	// Main container
	rw.Container = container.NewBorder(
		widget.NewLabel("Available rooms:"),
		buttonBar,
		nil, nil,
		container.NewScroll(rw.RoomList),
	)

	return rw.Container
}

// updateRoomList updates the room list in the interface
func (rw *RoomWindow) updateRoomList(rooms []models.Room) {
	rw.Rooms = rooms
	rw.RoomList.Refresh()
	rw.selectedRoomIndex = -1 // Resets the selected index
	rw.JoinBtn.Disable()      // Resets the join button state
}

// refreshRoomList requests a room list update from the server
func (rw *RoomWindow) refreshRoomList() {
	// Check if the network manager is ready
	if rw.UI.VPN.NetworkManager == nil {
		log.Println("Error: NetworkManager has not been initialized")
		return
	}

	// Try to update the room list in a separate goroutine
	go func() {
		// Try to connect if not already connected
		if !rw.UI.VPN.NetworkManager.IsConnected {
			err := rw.UI.VPN.NetworkManager.Connect()
			if err != nil {
				// Log the error
				log.Printf("Connection error: %v", err)

				// Use fyne.Do to execute code in the main UI thread
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Connection Error",
					Content: "Could not connect to the signaling server",
				})
				return
			}
		}

		// Perform the network operation
		err := rw.UI.VPN.NetworkManager.GetRoomList()
		if err != nil {
			log.Printf("Error getting room list: %v", err)

			// Use fyne.Do to execute code in the main UI thread
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Error",
				Content: "Could not update the room list",
			})
		}
	}()
}

// showCreateRoomDialog displays the dialog to create a new room
func (rw *RoomWindow) showCreateRoomDialog() {
	// Check if connected to the backend
	if !rw.UI.VPN.NetworkManager.IsConnected {
		// Try to connect to the server in a separate goroutine
		go func() {
			err := rw.UI.VPN.NetworkManager.Connect()
			// Return to the main thread to update the UI
			if err != nil {
				// Use SendNotification to display an error notification
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Connection Error",
					Content: "Could not connect to the signaling server",
				})
			} else {
				// If successfully connected, use the main window to show the dialog
				// Back to the main thread
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Connection Established",
					Content: "Connection successfully established",
				})
				// We need to use the main thread for UI
				// Since we can't use Driver().Run, we'll try to create the dialog
				// the next time the user clicks the button
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Ready",
					Content: "Click Create Room again to continue",
				})
			}
		}()
	} else {
		// Already connected, continue normally
		rw.showCreateRoomDialogAfterConnect()
	}
}

// showCreateRoomDialogAfterConnect displays the dialog after confirming server connection
func (rw *RoomWindow) showCreateRoomDialogAfterConnect() {
	// Check if the user already has a public key
	if rw.UI.VPN.PublicKeyPEM == "" {
		// If there is no public key, try to generate it
		err := rw.UI.VPN.loadOrGenerateKeys()
		if err != nil {
			rw.UI.ShowMessage("Error", "Could not create or load your identification keys: "+err.Error())
			return
		}
	}

	// Implement room creation dialog
	roomNameEntry := widget.NewEntry()
	roomPasswordEntry := widget.NewPasswordEntry()

	// Limit the password to only numbers with a maximum of 4 digits
	roomPasswordEntry.Validator = func(s string) error {
		if len(s) > 4 {
			return fmt.Errorf("password must be at most 4 characters")
		}
		for _, r := range s {
			if r < '0' || r > '9' {
				return fmt.Errorf("password must contain only numbers")
			}
		}
		return nil
	}

	// Limit input to 4 characters in real-time
	roomPasswordEntry.OnChanged = func(s string) {
		if len(s) > 4 {
			roomPasswordEntry.SetText(s[:4])
		}
	}

	content := container.NewVBox(
		widget.NewLabel("Room Name:"),
		roomNameEntry,
		widget.NewLabel("Password (4 digits):"),
		roomPasswordEntry,
		widget.NewLabel("Note: Each user can create only one room.\nIf you have already created a room, you cannot create another one."),
	)

	// Create the modal popup
	popup := widget.NewModalPopUp(
		container.NewVBox(
			content,
			container.NewHBox(
				widget.NewButton("Cancel", func() {
					if rw.Window.Canvas().Overlays().Top() != nil {
						rw.Window.Canvas().Overlays().Top().Hide()
					}
				}),
				widget.NewButton("Create", func() {
					// Validate fields
					if roomNameEntry.Text == "" || roomPasswordEntry.Text == "" {
						rw.UI.ShowMessage("Error", "Please fill in all fields")
						return
					}

					// Validate password length
					if len(roomPasswordEntry.Text) < 4 {
						rw.UI.ShowMessage("Error", "The password must be exactly 4 digits")
						return
					}

					// Create room in a separate goroutine
					go func() {
						err := rw.UI.VPN.NetworkManager.CreateRoom(roomNameEntry.Text, roomPasswordEntry.Text)
						if err != nil {
							// Log the error and show a notification
							log.Printf("Error creating room: %v", err)
							fyne.CurrentApp().SendNotification(&fyne.Notification{
								Title:   "Error",
								Content: "Could not create the room",
							})
						} else {
							// Success - just update the list in the main thread
							// Since we can't use Driver().Run directly,
							// we use SendNotification and will have the client click Refresh
							fyne.CurrentApp().SendNotification(&fyne.Notification{
								Title:   "Success",
								Content: "Room created successfully! Click Refresh List.",
							})
						}
					}()
				}),
			),
		),
		rw.Window.Canvas(),
	)

	popup.Show()
}

// showJoinRoomDialog displays the dialog to join a room
func (rw *RoomWindow) showJoinRoomDialog() {
	// Check if a room has been selected
	if rw.selectedRoomIndex < 0 || rw.selectedRoomIndex >= len(rw.Rooms) {
		rw.UI.ShowMessage("Error", "Please select a room first")
		return
	}

	selectedRoom := rw.Rooms[rw.selectedRoomIndex]
	roomPasswordEntry := widget.NewPasswordEntry()

	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Join room: %s", selectedRoom.Name)),
		widget.NewLabel("Password:"),
		roomPasswordEntry,
	)

	// Create the modal popup
	popup := widget.NewModalPopUp(
		container.NewVBox(
			content,
			container.NewHBox(
				widget.NewButton("Cancel", func() {
					if rw.Window.Canvas().Overlays().Top() != nil {
						rw.Window.Canvas().Overlays().Top().Hide()
					}
				}),
				widget.NewButton("Join", func() {
					// Validate fields
					if roomPasswordEntry.Text == "" {
						rw.UI.ShowMessage("Error", "Please enter the room password")
						return
					}

					// Join the room
					err := rw.UI.VPN.NetworkManager.JoinRoom(selectedRoom.ID, roomPasswordEntry.Text)
					if err != nil {
						rw.UI.ShowMessage("Error", "Could not join the room: "+err.Error())
					} else {
						// Update connection status
						rw.UI.VPN.CurrentRoom = selectedRoom.ID
						rw.UI.VPN.IsConnected = true
						rw.UI.updatePowerButtonState()
					}
					if rw.Window.Canvas().Overlays().Top() != nil {
						rw.Window.Canvas().Overlays().Top().Hide()
					}
				}),
			),
		),
		rw.Window.Canvas(),
	)

	popup.Show()
}
