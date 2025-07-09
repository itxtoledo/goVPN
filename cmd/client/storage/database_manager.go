package storage

import (
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Network represents a VPN network
type Network struct {
	ID            string
	Name          string
	LastConnected time.Time
}
