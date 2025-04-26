package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// loadEnvFile attempts to load environment variables from a .env file if it exists
// Logic: Try to load environment variables from .env file, but continue if not found
func loadEnvFile() {
	// Try to load from .env file
	err := godotenv.Load()
	if err != nil {
		// Only log at debug level since the .env file might not exist, which is normal
		log.Printf("No .env file found or error loading .env file: %v", err)
	} else {
		log.Println("Loaded environment variables from .env file")
	}
}

// Config structure to hold our server configuration
type Config struct {
	Port               string
	AllowAllOrigins    bool
	MaxRooms           int
	MaxClientsPerRoom  int
	LogLevel           string
	IdleTimeout        time.Duration
	PingInterval       time.Duration
	ReadBufferSize     int
	WriteBufferSize    int
	SupabaseURL        string
	SupabaseKey        string
	SupabaseRoomsTable string
	CleanupInterval    time.Duration
	RoomExpiryDays     int
}

// loadConfig loads configuration from environment variables and command-line flags
// Logic: Set default values and override with environment variables when available
func loadConfig() Config {
	// First try to load environment variables from a .env file
	loadEnvFile()

	// Define and parse command-line flags
	portFlag := flag.String("port", "", "Port number to listen on")
	allowOriginsFlag := flag.Bool("allow-all-origins", false, "Allow all origins for WebSocket connections")

	flag.Parse()

	// Get environment variables with fallbacks
	config := Config{
		Port:               getEnv("PORT", "8080"),
		AllowAllOrigins:    getEnvBool("ALLOW_ALL_ORIGINS", true),
		MaxRooms:           getEnvInt("MAX_ROOMS", 100),
		MaxClientsPerRoom:  getEnvInt("MAX_CLIENTS_PER_ROOM", 10),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		IdleTimeout:        time.Second * time.Duration(getEnvInt("IDLE_TIMEOUT_SECONDS", 60)),
		PingInterval:       time.Second * time.Duration(getEnvInt("PING_INTERVAL_SECONDS", 30)),
		ReadBufferSize:     getEnvInt("READ_BUFFER_SIZE", 1024),
		WriteBufferSize:    getEnvInt("WRITE_BUFFER_SIZE", 1024),
		SupabaseURL:        getEnv("SUPABASE_URL", ""),
		SupabaseKey:        getEnv("SUPABASE_KEY", ""),
		SupabaseRoomsTable: getEnv("SUPABASE_ROOMS_TABLE", "rooms"),
		CleanupInterval:    time.Hour * time.Duration(getEnvInt("CLEANUP_INTERVAL_HOURS", 24)),
		RoomExpiryDays:     getEnvInt("ROOM_EXPIRY_DAYS", 30),
	}

	// Override with command-line flags if provided
	if *portFlag != "" {
		config.Port = *portFlag
	}

	if *allowOriginsFlag {
		config.AllowAllOrigins = true
	}

	log.Printf("Server configuration loaded: %+v", config)
	return config
}

// Helper functions for environment variables

// getEnv gets an environment variable or returns a default value
// Logic: Retrieve environment variable value or use fallback if not set
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// getEnvInt gets an integer environment variable or returns a default value
// Logic: Convert environment variable to integer or use fallback if conversion fails
func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	intVal, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Warning: Could not parse %s as integer: %v. Using default: %d", key, err, fallback)
		return fallback
	}
	return intVal
}

// getEnvBool gets a boolean environment variable or returns a default value
// Logic: Convert environment variable to boolean considering multiple true values
func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	value = strings.ToLower(value)
	return value == "true" || value == "1" || value == "yes" || value == "y"
}

// RunServer starts the VPN server on the specified port
// Logic: Initialize the server and start listening for connections
func RunServer() {
	// Load configuration from environment variables and command-line flags
	config := loadConfig()

	// Check if Supabase configuration is provided
	if config.SupabaseURL == "" || config.SupabaseKey == "" {
		log.Fatal("Supabase URL and API key are required. Please set SUPABASE_URL and SUPABASE_KEY environment variables")
	}

	// Create and start the WebSocket server
	server, err := NewWebSocketServer(config)
	if err != nil {
		log.Fatalf("Failed to create WebSocket server: %v", err)
	}

	// Start the server (this will block until the server is stopped)
	log.Fatal(server.Start(config.Port))
}

// main is the entry point for the application
// Logic: Start the server
func main() {
	RunServer()
}
