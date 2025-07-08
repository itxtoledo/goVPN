package storage

import (
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Room represents a VPN room
type Room struct {
	ID            string
	Name          string
	LastConnected time.Time
}
