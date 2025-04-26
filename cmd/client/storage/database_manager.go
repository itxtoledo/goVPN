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
