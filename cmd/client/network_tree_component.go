// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/network_tree_component.go
package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// NetworkTreeComponent represents the networks list with categories
type NetworkTreeComponent struct {
	ui          *UIManager
	NetworkList *widget.List  // Changed from tree to list for better stability
	rooms       []models.Room // All rooms (local and remote)
	localRooms  map[string]models.Room
	container   *fyne.Container
}

// NewNetworkTreeComponent creates a new network tree component
func NewNetworkTreeComponent(ui *UIManager) *NetworkTreeComponent {
	ntc := &NetworkTreeComponent{
		ui:         ui,
		rooms:      []models.Room{},
		localRooms: make(map[string]models.Room),
	}

	// Create a list instead of a tree for more reliable display
	ntc.NetworkList = widget.NewList(
		// Length function
		func() int {
			// Return the number of items (categories + rooms)
			count := 2 // Always have "Recent Networks" and "Available Networks" headers
			count += len(ntc.localRooms)

			// Count available (non-local) rooms
			for _, room := range ntc.rooms {
				if _, exists := ntc.localRooms[room.ID]; !exists {
					count++
				}
			}

			return count
		},
		// Create item UI
		func() fyne.CanvasObject {
			// Create a template for list items with just an icon and label
			return container.NewHBox(
				widget.NewIcon(theme.HomeIcon()),
				widget.NewLabel("Template Item"),
			)
		},
		// Update item UI
		func(id widget.ListItemID, item fyne.CanvasObject) {
			// Convert to container and access elements
			hBox := item.(*fyne.Container)
			if len(hBox.Objects) < 2 {
				return
			}
			icon := hBox.Objects[0].(*widget.Icon)
			label := hBox.Objects[1].(*widget.Label)

			// Get the appropriate item based on id
			if id == 0 {
				// Recent Networks header
				icon.SetResource(theme.FolderOpenIcon())
				label.SetText("Recent Networks")
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else if id == 1+len(ntc.localRooms) {
				// Available Networks header
				icon.SetResource(theme.FolderOpenIcon())
				label.SetText("Available Networks")
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else if id <= len(ntc.localRooms) {
				// Recent network item
				var i int = 0
				for _, room := range ntc.getSortedLocalRooms() {
					if i == id-1 { // -1 because we have a header
						icon.SetResource(theme.ComputerIcon())
						label.SetText(room.Name)
						label.TextStyle = fyne.TextStyle{} // Reset any style
						break
					}
					i++
				}
			} else {
				// Available network item
				var i int = 0
				for _, room := range ntc.getAvailableRooms() {
					if i == id-(2+len(ntc.localRooms)) { // -2 for both headers, -len for local rooms
						icon.SetResource(theme.StorageIcon())
						label.SetText(room.Name)
						label.TextStyle = fyne.TextStyle{} // Reset any style
						break
					}
					i++
				}
			}
		},
	)

	// Create scrollable container with the list
	scrollContainer := container.NewScroll(ntc.NetworkList)
	ntc.container = container.NewMax(scrollContainer)

	// Set a reasonable min size
	// ntc.NetworkList.MinSize = fyne.NewSize(280, 300)

	// Register callback to update when room list changes
	if ui != nil && ui.VPN != nil && ui.VPN.NetworkManager != nil {
		ui.VPN.NetworkManager.OnRoomListUpdate = func(rooms []models.Room) {
			ntc.updateRooms(rooms)
		}
	}

	// Load initial rooms without attempting to connect
	ntc.loadLocalRooms()

	return ntc
}

// GetContainer returns the component's container
func (ntc *NetworkTreeComponent) GetContainer() *fyne.Container {
	return ntc.container
}

// loadLocalRooms loads rooms from local database
func (ntc *NetworkTreeComponent) loadLocalRooms() {
	// Ensure the VPN client and NetworkManager are available
	if ntc.ui == nil || ntc.ui.VPN == nil || ntc.ui.VPN.NetworkManager == nil {
		log.Printf("Cannot load local rooms: UI or NetworkManager is nil")
		return
	}

	localRooms, err := ntc.ui.VPN.NetworkManager.loadLocalRooms()
	if err != nil {
		log.Printf("Error loading local rooms: %v", err)
		return
	}

	// Update local rooms map
	ntc.localRooms = make(map[string]models.Room)
	for _, room := range localRooms {
		ntc.localRooms[room.ID] = room
	}

	// Update the component
	ntc.updateRooms(nil)
}

// updateRooms updates the list with new rooms from server
func (ntc *NetworkTreeComponent) updateRooms(serverRooms []models.Room) {
	// Safety check
	if ntc.NetworkList == nil {
		log.Printf("Warning: NetworkList is nil in updateRooms")
		return
	}

	// Combine local and server rooms
	allRooms := []models.Room{}

	// Add local rooms first
	for _, room := range ntc.localRooms {
		allRooms = append(allRooms, room)
	}

	// Add server rooms that aren't already in local
	if serverRooms != nil {
		for _, room := range serverRooms {
			if _, exists := ntc.localRooms[room.ID]; !exists {
				allRooms = append(allRooms, room)
			}
		}
	}

	// Update the component state
	ntc.rooms = allRooms

	// Update the list
	ntc.NetworkList.Refresh()
}

// getSortedLocalRooms returns local rooms sorted by last used time
func (ntc *NetworkTreeComponent) getSortedLocalRooms() []models.Room {
	rooms := []models.Room{}

	// Collect all local rooms
	for _, room := range ntc.localRooms {
		rooms = append(rooms, room)
	}

	// We could add sorting logic here if we tracked last used time
	// For now, let's leave them as they come from the database which
	// should already have some order

	return rooms
}

// getAvailableRooms returns rooms that are not in local storage
func (ntc *NetworkTreeComponent) getAvailableRooms() []models.Room {
	rooms := []models.Room{}

	// Find rooms that aren't in local storage
	for _, room := range ntc.rooms {
		if _, exists := ntc.localRooms[room.ID]; !exists {
			rooms = append(rooms, room)
		}
	}

	return rooms
}

// Refresh explicitly refreshes the network list
func (ntc *NetworkTreeComponent) Refresh() {
	if ntc.NetworkList != nil {
		ntc.NetworkList.Refresh()
	}
}
