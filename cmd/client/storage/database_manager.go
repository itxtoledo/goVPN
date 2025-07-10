package storage

import (
	_ "github.com/mattn/go-sqlite3"
)

// DatabaseManager gerencia as operações do banco de dados SQLite
type DatabaseManager struct {
	dbPath string
}
