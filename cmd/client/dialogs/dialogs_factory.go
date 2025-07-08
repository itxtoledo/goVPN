package dialogs

import (
	"time"

	"fyne.io/fyne/v2"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/storage"
	"github.com/itxtoledo/govpn/libs/models"
)

// NetworkManagerInterface defines the necessary methods from NetworkManager for dialogs
type NetworkManagerInterface interface {
	CreateRoom(name, password string) (*models.CreateRoomResponse, error)
	JoinRoom(roomID, password, username string) (*models.JoinRoomResponse, error)
	GetRoomID() string
	GetRealtimeData() *data.RealtimeDataLayer
}

// NewCreateRoomDialog creates a new dialog for creating a room
func NewCreateRoomDialog(networkManager NetworkManagerInterface, mainWindow fyne.Window, username string) *RoomDialog {
	return NewRoomDialog(
		mainWindow,
		networkManager.CreateRoom,
		func(roomID string, name string, password string) error {
			// This is where you'd save the room to local storage if needed
			// For now, we'll just add it to the RealtimeDataLayer
			room := &storage.Room{
				ID:            roomID,
				Name:          name,
				LastConnected: time.Now(),
			}
			networkManager.GetRealtimeData().AddRoom(room)
			return nil
		},
		func() string { return networkManager.GetRoomID() },
		username,
		ValidatePassword,
		ConfigurePasswordEntry,
	)
}

// NewJoinRoomDialog creates a new dialog for joining a room
func NewJoinRoomDialog(networkManager NetworkManagerInterface, mainWindow fyne.Window, username string) *JoinDialog {
	return NewJoinDialog(
		mainWindow,
		networkManager.JoinRoom,
		func(roomID string, name string, password string) error {
			// This is where you'd save the room to local storage if needed
			// For now, we'll just add it to the RealtimeDataLayer
			room := &storage.Room{
				ID:            roomID,
				Name:          name,
				LastConnected: time.Now(),
			}
			networkManager.GetRealtimeData().AddRoom(room)
			return nil
		},
		username,
		ValidatePassword,
		ConfigurePasswordEntry,
	)
}
