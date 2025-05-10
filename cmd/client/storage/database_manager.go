package storage

import (
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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
