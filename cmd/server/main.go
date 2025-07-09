package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/itxtoledo/govpn/cmd/server/logger"
	"github.com/joho/godotenv"
)

// Config holds the configuration for the WebSocket server
type Config struct {
	Port                  string        // Port to listen on
	SupabaseURL           string        // URL of the Supabase instance
	SupabaseKey           string        // API key for Supabase
	SupabaseNetworksTable string        // Name of the networks table in Supabase
	ReadBufferSize        int           // Size of the read buffer for WebSocket connections
	WriteBufferSize       int           // Size of the write buffer for WebSocket connections
	MaxClientsPerNetwork  int           // Maximum number of clients allowed in a network
	NetworkExpiryDays     int           // Number of days after which inactive networks are deleted
	AllowAllOrigins       bool          // Whether to allow all origins for WebSocket connections
	CleanupInterval       time.Duration // Interval at which to clean up stale networks
	LogLevel              string        // Log level (debug, info, warn, error)
	ShutdownTimeout       time.Duration // Timeout for graceful shutdown
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
		// Will initialize logger first with default level before accessing config
		logger.Init(logger.InfoLevel)
		logger.Warn("Could not load .env file", "error", err)
	}

	// Default configuration
	cfg := Config{
		Port:                  getEnv("PORT", "8080"),
		SupabaseURL:           getEnv("SUPABASE_URL", ""),
		SupabaseKey:           getEnv("SUPABASE_KEY", ""),
		SupabaseNetworksTable: getEnv("SUPABASE_NETWORKS_TABLE", "networks"),
		ReadBufferSize:        1024,
		WriteBufferSize:       1024,
		MaxClientsPerNetwork:  50,
		NetworkExpiryDays:     7,
		AllowAllOrigins:       true,
		CleanupInterval:       24 * time.Hour, // Run cleanup once a day
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout:       15 * time.Second, // Default timeout for graceful shutdown
	}

	// Initialize logger with configured log level
	logger.Init(logger.LogLevel(cfg.LogLevel))

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

	if maxClients := getEnv("MAX_CLIENTS_PER_NETWORK", ""); maxClients != "" {
		if max, err := strconv.Atoi(maxClients); err == nil {
			cfg.MaxClientsPerNetwork = max
		}
	}

	if expiryDays := getEnv("NETWORK_EXPIRY_DAYS", ""); expiryDays != "" {
		if days, err := strconv.Atoi(expiryDays); err == nil {
			cfg.NetworkExpiryDays = days
		}
	}

	if cleanupInterval := getEnv("CLEANUP_INTERVAL_HOURS", ""); cleanupInterval != "" {
		if hours, err := strconv.Atoi(cleanupInterval); err == nil {
			cfg.CleanupInterval = time.Duration(hours) * time.Hour
		}
	}

	// Parse shutdown timeout
	if shutdownTimeout := getEnv("SHUTDOWN_TIMEOUT_SECONDS", "2"); shutdownTimeout != "" {
		if seconds, err := strconv.Atoi(shutdownTimeout); err == nil && seconds > 0 {
			cfg.ShutdownTimeout = time.Duration(seconds) * time.Second
		}
	}

	// Parse boolean environment variables
	if allowAllOrigins := getEnv("ALLOW_ALL_ORIGINS", ""); allowAllOrigins != "" {
		cfg.AllowAllOrigins = allowAllOrigins == "true"
	}

	// Create new WebSocket server with the configuration
	server, err := NewWebSocketServer(cfg)
	if err != nil {
		logger.Fatal("Failed to create WebSocket server", "error", err)
	}

	// Start the server
	logger.Info("Starting WebSocket server", "port", cfg.Port)
	err = server.Start(cfg.Port)
	if err != nil {
		logger.Fatal("Failed to start server", "error", err)
	}

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a termination signal
	sig := <-signalChan
	logger.Info("Received signal, starting graceful shutdown", "signal", sig)

	// Set an appropriate restart message based on signal
	restartInfo := "Server will be back soon."
	if sig == syscall.SIGTERM {
		restartInfo = "Server is being restarted for maintenance."
	}

	// Initiate graceful shutdown
	server.InitiateGracefulShutdown(cfg.ShutdownTimeout, restartInfo)

	// Wait for shutdown to complete
	server.WaitForShutdown()
	logger.Info("Server has shut down gracefully")

	// Ensure all logs are flushed
	logger.Sync()
}
