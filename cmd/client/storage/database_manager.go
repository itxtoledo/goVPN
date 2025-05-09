package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DatabaseManager handles all database interactions
type DatabaseManager struct {
	DB *sql.DB
}

// Room represents a VPN room
type Room struct {
	ID            string
	Name          string
	Password      string
	LastConnected time.Time
}

// JoinedRoom represents a room the user has joined
type JoinedRoom struct {
	ID            int64
	RoomID        string
	RoomName      string
	JoinedAt      time.Time
	LastConnected time.Time
	IsConnected   bool
}

// NewDatabaseManager creates and initializes a new database manager
func NewDatabaseManager() (*DatabaseManager, error) {
	// Create the user data directory if it doesn't exist
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	appDir := filepath.Join(userConfigDir, "goVPN")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, err
	}

	// Connect to SQLite database
	dbPath := filepath.Join(appDir, "govpn.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS rooms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			password TEXT NOT NULL,
			last_connected TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		
		CREATE TABLE IF NOT EXISTS keys (
			id TEXT PRIMARY KEY,
			private_key TEXT NOT NULL,
			public_key TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		
		CREATE TABLE IF NOT EXISTS joined_rooms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id TEXT NOT NULL,
			room_name TEXT NOT NULL,
			joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_connected TIMESTAMP,
			is_connected INTEGER DEFAULT 0,
			UNIQUE(room_id),
			FOREIGN KEY (room_id) REFERENCES rooms(id)
		);
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DatabaseManager{DB: db}, nil
}

// Close closes the database connection
func (dm *DatabaseManager) Close() error {
	if dm.DB != nil {
		return dm.DB.Close()
	}
	return nil
}

// SaveSetting saves a setting to the database
func (dm *DatabaseManager) SaveSetting(key, value string) error {
	// Use INSERT OR REPLACE to handle both new settings and updates
	_, err := dm.DB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

// GetSetting retrieves a setting from the database
func (dm *DatabaseManager) GetSetting(key string) (string, error) {
	var value string
	err := dm.DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	return value, err
}

// LoadSignalServer loads the signal server configuration
func (dm *DatabaseManager) LoadSignalServer() (string, error) {
	return dm.GetSetting("signal_server")
}

// LoadStunServer loads the STUN server configuration
func (dm *DatabaseManager) LoadStunServer() (string, error) {
	return dm.GetSetting("stun_server")
}

// SaveKeys stores Ed25519 keys in the database
func (dm *DatabaseManager) SaveKeys(privateKey, publicKey string) error {
	_, err := dm.DB.Exec("INSERT OR REPLACE INTO keys (id, private_key, public_key) VALUES ('user_key', ?, ?)",
		privateKey, publicKey)
	return err
}

// LoadKeys loads Ed25519 keys from the database
func (dm *DatabaseManager) LoadKeys() (privateKey string, publicKey string, err error) {
	err = dm.DB.QueryRow("SELECT private_key, public_key FROM keys WHERE id = 'user_key' LIMIT 1").
		Scan(&privateKey, &publicKey)
	return privateKey, publicKey, err
}

// SaveRSAKeys is kept for backward compatibility, now uses SaveKeys
func (dm *DatabaseManager) SaveRSAKeys(privateKey, publicKey string) error {
	return dm.SaveKeys(privateKey, publicKey)
}

// LoadRSAKeys is kept for backward compatibility, now uses LoadKeys
func (dm *DatabaseManager) LoadRSAKeys() (privateKey string, publicKey string, err error) {
	return dm.LoadKeys()
}

// SaveRoom saves a room to the database
func (dm *DatabaseManager) SaveRoom(id, name, password string) error {
	_, err := dm.DB.Exec("INSERT OR REPLACE INTO rooms (id, name, password, last_connected) VALUES (?, ?, ?, ?)",
		id, name, password, time.Now())
	return err
}

// GetRooms retrieves all saved rooms
func (dm *DatabaseManager) GetRooms() ([]Room, error) {
	rows, err := dm.DB.Query("SELECT id, name, password, last_connected FROM rooms ORDER BY last_connected DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []Room

	for rows.Next() {
		var room Room
		if err := rows.Scan(&room.ID, &room.Name, &room.Password, &room.LastConnected); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	return rooms, rows.Err()
}

// UpdateRoomConnection updates the last connection time for a room
func (dm *DatabaseManager) UpdateRoomConnection(roomID string) error {
	_, err := dm.DB.Exec("UPDATE rooms SET last_connected = ? WHERE id = ?", time.Now(), roomID)
	return err
}

// DeleteRoom remove uma sala do banco de dados local pelo ID
func (dm *DatabaseManager) DeleteRoom(roomID string) error {
	_, err := dm.DB.Exec("DELETE FROM rooms WHERE id = ?", roomID)
	return err
}

// SaveJoinedRoom adds or updates a joined room record
func (dm *DatabaseManager) SaveJoinedRoom(roomID, roomName string) error {
	_, err := dm.DB.Exec(`INSERT OR REPLACE INTO joined_rooms (room_id, room_name, joined_at, last_connected, is_connected) 
		VALUES (?, ?, datetime('now'), datetime('now'), 0)`, roomID, roomName)
	return err
}

// UpdateJoinedRoomConnection updates the connection status and timestamp for a joined room
func (dm *DatabaseManager) UpdateJoinedRoomConnection(roomID string, isConnected bool) error {
	connected := 0
	if isConnected {
		connected = 1
	}
	_, err := dm.DB.Exec("UPDATE joined_rooms SET last_connected = datetime('now'), is_connected = ? WHERE room_id = ?",
		connected, roomID)
	return err
}

// RemoveJoinedRoom removes a room from the joined_rooms table
func (dm *DatabaseManager) RemoveJoinedRoom(roomID string) error {
	_, err := dm.DB.Exec("DELETE FROM joined_rooms WHERE room_id = ?", roomID)
	return err
}

// GetJoinedRooms retrieves all rooms the user has joined
func (dm *DatabaseManager) GetJoinedRooms() ([]JoinedRoom, error) {
	rows, err := dm.DB.Query(`SELECT id, room_id, room_name, joined_at, last_connected, is_connected 
		FROM joined_rooms ORDER BY last_connected DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []JoinedRoom
	for rows.Next() {
		var room JoinedRoom
		var isConnected int
		if err := rows.Scan(&room.ID, &room.RoomID, &room.RoomName, &room.JoinedAt, &room.LastConnected, &isConnected); err != nil {
			return nil, err
		}
		room.IsConnected = isConnected != 0
		rooms = append(rooms, room)
	}

	return rooms, rows.Err()
}

// GetConnectedRooms retrieves all rooms the user is currently connected to
func (dm *DatabaseManager) GetConnectedRooms() ([]JoinedRoom, error) {
	rows, err := dm.DB.Query(`SELECT id, room_id, room_name, joined_at, last_connected, is_connected 
		FROM joined_rooms WHERE is_connected = 1 ORDER BY last_connected DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []JoinedRoom
	for rows.Next() {
		var room JoinedRoom
		var isConnected int
		if err := rows.Scan(&room.ID, &room.RoomID, &room.RoomName, &room.JoinedAt, &room.LastConnected, &isConnected); err != nil {
			return nil, err
		}
		room.IsConnected = isConnected != 0
		rooms = append(rooms, room)
	}

	return rooms, rows.Err()
}

// IsJoinedRoom checks if a user has joined a specific room
func (dm *DatabaseManager) IsJoinedRoom(roomID string) (bool, error) {
	var count int
	err := dm.DB.QueryRow("SELECT COUNT(*) FROM joined_rooms WHERE room_id = ?", roomID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetJoinedRoom gets a specific joined room by roomID
func (dm *DatabaseManager) GetJoinedRoom(roomID string) (JoinedRoom, error) {
	var room JoinedRoom
	var isConnected int
	err := dm.DB.QueryRow(`SELECT id, room_id, room_name, joined_at, last_connected, is_connected 
		FROM joined_rooms WHERE room_id = ?`, roomID).
		Scan(&room.ID, &room.RoomID, &room.RoomName, &room.JoinedAt, &room.LastConnected, &isConnected)
	room.IsConnected = isConnected != 0
	return room, err
}
