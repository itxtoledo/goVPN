package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the configuration for the WebSocket server
type Config struct {
	Port               string        // Port to listen on
	SupabaseURL        string        // URL of the Supabase instance
	SupabaseKey        string        // API key for Supabase
	SupabaseRoomsTable string        // Name of the rooms table in Supabase
	ReadBufferSize     int           // Size of the read buffer for WebSocket connections
	WriteBufferSize    int           // Size of the write buffer for WebSocket connections
	MaxClientsPerRoom  int           // Maximum number of clients allowed in a room
	RoomExpiryDays     int           // Number of days after which inactive rooms are deleted
	AllowAllOrigins    bool          // Whether to allow all origins for WebSocket connections
	CleanupInterval    time.Duration // Interval at which to clean up stale rooms
	LogLevel           string        // Log level (debug, info, warn, error)
}

// getEnv retrieves the value of an environment variable, prioritizing the .env file
// If the variable is not found in either source, it returns the provided default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func main() {
	// Load .env file if present
	envPath := filepath.Join(".", ".env")
	err := godotenv.Load(envPath)
	if err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Default configuration
	cfg := Config{
		Port:               getEnv("PORT", "8080"),
		SupabaseURL:        getEnv("SUPABASE_URL", ""),
		SupabaseKey:        getEnv("SUPABASE_KEY", ""),
		SupabaseRoomsTable: getEnv("SUPABASE_ROOMS_TABLE", "rooms"),
		ReadBufferSize:     1024,
		WriteBufferSize:    1024,
		MaxClientsPerRoom:  50,
		RoomExpiryDays:     7,
		AllowAllOrigins:    true,
		CleanupInterval:    24 * time.Hour, // Run cleanup once a day
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}

	// Parse numeric environment variables
	if readSize := getEnv("READ_BUFFER_SIZE", ""); readSize != "" {
		if size, err := strconv.Atoi(readSize); err == nil {
			cfg.ReadBufferSize = size
		}
	}

	if writeSize := getEnv("WRITE_BUFFER_SIZE", ""); writeSize != "" {
		if size, err := strconv.Atoi(writeSize); err == nil {
			cfg.WriteBufferSize = size
		}
	}

	if maxClients := getEnv("MAX_CLIENTS_PER_ROOM", ""); maxClients != "" {
		if max, err := strconv.Atoi(maxClients); err == nil {
			cfg.MaxClientsPerRoom = max
		}
	}

	if expiryDays := getEnv("ROOM_EXPIRY_DAYS", ""); expiryDays != "" {
		if days, err := strconv.Atoi(expiryDays); err == nil {
			cfg.RoomExpiryDays = days
		}
	}

	if cleanupInterval := getEnv("CLEANUP_INTERVAL_HOURS", ""); cleanupInterval != "" {
		if hours, err := strconv.Atoi(cleanupInterval); err == nil {
			cfg.CleanupInterval = time.Duration(hours) * time.Hour
		}
	}

	// Parse boolean environment variables
	if allowAllOrigins := getEnv("ALLOW_ALL_ORIGINS", ""); allowAllOrigins != "" {
		cfg.AllowAllOrigins = allowAllOrigins == "true"
	}

	// Create new WebSocket server with the configuration
	server, err := NewWebSocketServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create WebSocket server: %v", err)
	}

	// Start the server
	log.Printf("Starting WebSocket server on port %s", cfg.Port)
	err = server.Start(cfg.Port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
