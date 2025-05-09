package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/itxtoledo/govpn/cmd/server/logger"
	"github.com/itxtoledo/govpn/libs/models"
	"github.com/supabase-community/supabase-go"
)

// SupabaseManager handles all Supabase database operations for the server
type SupabaseManager struct {
	client     *supabase.Client
	roomsTable string
	logLevel   string
}

// NewSupabaseManager creates a new instance of the Supabase manager
func NewSupabaseManager(supabaseURL, supabaseKey, roomsTable, logLevel string) (*SupabaseManager, error) {
	client, err := supabase.NewClient(supabaseURL, supabaseKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Supabase client: %w", err)
	}

	return &SupabaseManager{
		client:     client,
		roomsTable: roomsTable,
		logLevel:   logLevel,
	}, nil
}

// CreateRoom inserts a new room into the Supabase database
func (sm *SupabaseManager) CreateRoom(room ServerRoom) error {
	roomData := map[string]interface{}{
		"id":          room.ID,
		"name":        room.Name,
		"password":    room.Password,
		"public_key":  room.PublicKeyB64,
		"created_at":  room.CreatedAt.Format(time.RFC3339),
		"last_active": room.LastActive.Format(time.RFC3339),
	}

	if sm.logLevel == "debug" {
		logger.Debug("Creating room in Supabase", "roomID", room.ID, "roomName", room.Name)
	}

	_, _, err := sm.client.From(sm.roomsTable).Insert(roomData, false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to create room in Supabase: %w", err)
	}

	return nil
}

// GetRoom fetches a room from the Supabase database by its ID
func (sm *SupabaseManager) GetRoom(roomID string) (ServerRoom, error) {
	var rooms []SupabaseRoom
	data, _, err := sm.client.From(sm.roomsTable).Select("*", "", false).Eq("id", roomID).Execute()
	if err != nil {
		return ServerRoom{}, fmt.Errorf("failed to fetch room from Supabase: %w", err)
	}

	if err := json.Unmarshal(data, &rooms); err != nil {
		return ServerRoom{}, fmt.Errorf("failed to parse room data: %w", err)
	}

	if len(rooms) == 0 {
		return ServerRoom{}, fmt.Errorf("room not found: %s", roomID)
	}

	dbRoom := rooms[0]

	// Create a ServerRoom from the SupabaseRoom data
	return ServerRoom{
		Room: models.Room{
			ID:       dbRoom.ID,
			Name:     dbRoom.Name,
			Password: dbRoom.Password,
		},
		PublicKeyB64: dbRoom.PublicKey,
		CreatedAt:    dbRoom.CreatedAt,
		LastActive:   dbRoom.LastActive,
	}, nil
}

// UpdateRoomActivity updates the last_active timestamp for a room
func (sm *SupabaseManager) UpdateRoomActivity(roomID string) error {
	now := time.Now().Format(time.RFC3339)
	updateData := map[string]interface{}{
		"last_active": now,
	}

	if sm.logLevel == "debug" {
		logger.Debug("Updating last_active for room", "roomID", roomID, "timestamp", now)
	}

	_, _, err := sm.client.From(sm.roomsTable).Update(updateData, "", "").Eq("id", roomID).Execute()
	if err != nil {
		return fmt.Errorf("failed to update room activity: %w", err)
	}

	return nil
}

// UpdateRoomName updates the name of a room
func (sm *SupabaseManager) UpdateRoomName(roomID, newName string) error {
	updateData := map[string]interface{}{
		"name":        newName,
		"last_active": time.Now().Format(time.RFC3339),
	}

	if sm.logLevel == "debug" {
		logger.Debug("Updating name for room", "roomID", roomID, "newName", newName)
	}

	_, _, err := sm.client.From(sm.roomsTable).Update(updateData, "", "").Eq("id", roomID).Execute()
	if err != nil {
		return fmt.Errorf("failed to update room name: %w", err)
	}

	return nil
}

// DeleteRoom removes a room from the Supabase database
func (sm *SupabaseManager) DeleteRoom(roomID string) error {
	if sm.logLevel == "debug" {
		logger.Debug("Deleting room from Supabase", "roomID", roomID)
	}

	_, _, err := sm.client.From(sm.roomsTable).Delete("", "").Eq("id", roomID).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}

	return nil
}

// GetRoomByPublicKey fetches a room by the owner's public key
func (sm *SupabaseManager) GetRoomByPublicKey(publicKey string) (ServerRoom, error) {
	var rooms []SupabaseRoom
	data, _, err := sm.client.From(sm.roomsTable).Select("*", "", false).Eq("public_key", publicKey).Execute()
	if err != nil {
		return ServerRoom{}, fmt.Errorf("failed to fetch room by public key: %w", err)
	}

	if err := json.Unmarshal(data, &rooms); err != nil {
		return ServerRoom{}, fmt.Errorf("failed to parse room data: %w", err)
	}

	if len(rooms) == 0 {
		return ServerRoom{}, fmt.Errorf("no room found for public key")
	}

	dbRoom := rooms[0]

	return ServerRoom{
		Room: models.Room{
			ID:       dbRoom.ID,
			Name:     dbRoom.Name,
			Password: dbRoom.Password,
		},
		PublicKeyB64: dbRoom.PublicKey,
		CreatedAt:    dbRoom.CreatedAt,
		LastActive:   dbRoom.LastActive,
	}, nil
}

// GetStaleRooms fetches rooms that have not been active for a specified period
func (sm *SupabaseManager) GetStaleRooms(expiryDays int) ([]SupabaseRoom, error) {
	expiryDuration := time.Hour * 24 * time.Duration(expiryDays)
	cutoffTime := time.Now().Add(-expiryDuration)
	cutoffTimeStr := cutoffTime.Format(time.RFC3339)

	if sm.logLevel == "debug" {
		logger.Debug("Fetching stale rooms", "cutoffTime", cutoffTimeStr, "expiryDays", expiryDays)
	}

	var staleRooms []SupabaseRoom
	data, _, err := sm.client.From(sm.roomsTable).Select("*", "", false).Lt("last_active", cutoffTimeStr).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stale rooms: %w", err)
	}

	if err := json.Unmarshal(data, &staleRooms); err != nil {
		return nil, fmt.Errorf("failed to parse stale rooms data: %w", err)
	}

	return staleRooms, nil
}

// RoomExists checks if a room exists with the given ID
func (sm *SupabaseManager) RoomExists(roomID string) (bool, error) {
	var rooms []map[string]interface{}
	data, _, err := sm.client.From(sm.roomsTable).Select("id", "", false).Eq("id", roomID).Execute()
	if err != nil {
		return false, fmt.Errorf("failed to check if room exists: %w", err)
	}

	if err := json.Unmarshal(data, &rooms); err != nil {
		return false, fmt.Errorf("failed to parse room data: %w", err)
	}

	return len(rooms) > 0, nil
}

// PublicKeyHasRoom checks if a public key already has an associated room
func (sm *SupabaseManager) PublicKeyHasRoom(publicKey string) (bool, string, error) {
	var rooms []map[string]interface{}
	data, _, err := sm.client.From(sm.roomsTable).Select("id", "", false).Eq("public_key", publicKey).Execute()
	if err != nil {
		return false, "", fmt.Errorf("failed to check if public key has room: %w", err)
	}

	if err := json.Unmarshal(data, &rooms); err != nil {
		return false, "", fmt.Errorf("failed to parse room data: %w", err)
	}

	if len(rooms) > 0 {
		return true, rooms[0]["id"].(string), nil
	}
	return false, "", nil
}

// UserRoom represents a record in the user_rooms table linking users to rooms
type UserRoom struct {
	ID            int       `json:"id"`
	RoomID        string    `json:"room_id"`
	PublicKey     string    `json:"public_key"`
	Username      string    `json:"username"`
	JoinedAt      time.Time `json:"joined_at"`
	LastConnected time.Time `json:"last_connected"`
	IsConnected   bool      `json:"is_connected"`
}

// AddUserToRoom adds a user to a room in the user_rooms table
func (sm *SupabaseManager) AddUserToRoom(roomID, publicKey, username string) error {
	userRoomData := map[string]interface{}{
		"room_id":        roomID,
		"public_key":     publicKey,
		"username":       username,
		"joined_at":      time.Now().Format(time.RFC3339),
		"last_connected": time.Now().Format(time.RFC3339),
		"is_connected":   true,
	}

	if sm.logLevel == "debug" {
		logger.Debug("Adding user to room", "roomID", roomID, "publicKey", publicKey)
	}

	_, _, err := sm.client.From("user_rooms").Insert(userRoomData, false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to add user to room in Supabase: %w", err)
	}

	return nil
}

// UpdateUserRoomConnection updates the connection status and last_connected timestamp for a user in a room
func (sm *SupabaseManager) UpdateUserRoomConnection(roomID, publicKey string, isConnected bool) error {
	updateData := map[string]interface{}{
		"last_connected": time.Now().Format(time.RFC3339),
		"is_connected":   isConnected,
	}

	if sm.logLevel == "debug" {
		logger.Debug("Updating user room connection", "roomID", roomID, "publicKey", publicKey, "isConnected", isConnected)
	}

	_, _, err := sm.client.From("user_rooms").Update(updateData, "", "").Eq("room_id", roomID).Eq("public_key", publicKey).Execute()
	if err != nil {
		return fmt.Errorf("failed to update user room connection: %w", err)
	}

	return nil
}

// RemoveUserFromRoom removes a user from a room
func (sm *SupabaseManager) RemoveUserFromRoom(roomID, publicKey string) error {
	if sm.logLevel == "debug" {
		logger.Debug("Removing user from room", "roomID", roomID, "publicKey", publicKey)
	}

	_, _, err := sm.client.From("user_rooms").Delete("", "").Eq("room_id", roomID).Eq("public_key", publicKey).Execute()
	if err != nil {
		return fmt.Errorf("failed to remove user from room: %w", err)
	}

	return nil
}

// GetRoomUsers gets all users for a specific room
func (sm *SupabaseManager) GetRoomUsers(roomID string) ([]UserRoom, error) {
	var userRooms []UserRoom
	data, _, err := sm.client.From("user_rooms").Select("*", "", false).Eq("room_id", roomID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get room users: %w", err)
	}

	if err := json.Unmarshal(data, &userRooms); err != nil {
		return nil, fmt.Errorf("failed to parse user room data: %w", err)
	}

	return userRooms, nil
}

// GetUserRooms gets all rooms a user has joined
func (sm *SupabaseManager) GetUserRooms(publicKey string) ([]UserRoom, error) {
	var userRooms []UserRoom
	data, _, err := sm.client.From("user_rooms").Select("*", "", false).Eq("public_key", publicKey).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get user rooms: %w", err)
	}

	if err := json.Unmarshal(data, &userRooms); err != nil {
		return nil, fmt.Errorf("failed to parse user room data: %w", err)
	}

	return userRooms, nil
}

// IsUserInRoom checks if a user is already in a room
func (sm *SupabaseManager) IsUserInRoom(roomID, publicKey string) (bool, error) {
	var userRooms []UserRoom
	data, _, err := sm.client.From("user_rooms").Select("id", "", false).Eq("room_id", roomID).Eq("public_key", publicKey).Execute()
	if err != nil {
		return false, fmt.Errorf("failed to check if user is in room: %w", err)
	}

	if err := json.Unmarshal(data, &userRooms); err != nil {
		return false, fmt.Errorf("failed to parse user room data: %w", err)
	}

	return len(userRooms) > 0, nil
}
