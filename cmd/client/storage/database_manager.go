package storage

import (
	"database/sql"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DatabaseManager gerencia as operações do banco de dados SQLite
type DatabaseManager struct {
	db     *sql.DB
	dbPath string
}

// NewDatabaseManager cria uma nova instância do gerenciador de banco de dados
func NewDatabaseManager(dataPath string) *DatabaseManager {
	dbFilePath := filepath.Join(dataPath, "govpn.db")
	return &DatabaseManager{
		dbPath: dbFilePath,
	}
}

// Open abre a conexão com o banco de dados
func (dm *DatabaseManager) Open() error {
	var err error
	dm.db, err = sql.Open("sqlite3", dm.dbPath)
	if err != nil {
		return err
	}

	// Cria a tabela de redes se não existir
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS networks (
		id TEXT NOT NULL PRIMARY KEY,
		name TEXT,
		last_connected DATETIME
	);
	`
	_, err = dm.db.Exec(sqlStmt)
	if err != nil {
		log.Printf("Error creating networks table: %v", err)
		return err
	}

	log.Printf("Database opened and initialized at: %s", dm.dbPath)
	return nil
}

// Close fecha a conexão com o banco de dados
func (dm *DatabaseManager) Close() error {
	if dm.db != nil {
		log.Printf("Closing database at: %s", dm.dbPath)
		return dm.db.Close()
	}
	return nil
}

// GetDB retorna a instância do banco de dados
func (dm *DatabaseManager) GetDB() *sql.DB {
	return dm.db
}