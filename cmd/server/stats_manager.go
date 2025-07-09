// filepath: /Computers/gustavotoledodesouza/Projects/fun/goVPN/cmd/server/stats_manager.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/itxtoledo/govpn/cmd/server/logger"
)

// StatsManager gerencia as estatísticas do servidor WebSocket
type StatsManager struct {
	stats  ServerStats
	mu     sync.RWMutex
	config Config
}

// ServerStats armazena várias métricas do servidor WebSocket
type ServerStats struct {
	StartTime            time.Time `json:"start_time"`             // Quando o servidor foi iniciado
	Uptime               string    `json:"uptime"`                 // Tempo de atividade legível
	ConnectionsTotal     int       `json:"connections_total"`      // Total de conexões desde o início
	ActiveConnections    int       `json:"active_connections"`     // Número atual de conexões ativas
	ActiveNetworks       int       `json:"active_networks"`        // Número de salas ativas
	PeakConnections      int       `json:"peak_connections"`       // Número máximo de conexões simultâneas
	PeakNetworks         int       `json:"peak_networks"`          // Número máximo de salas simultâneas
	MessagesProcessed    int64     `json:"messages_processed"`     // Número de mensagens processadas
	Version              string    `json:"version"`                // Versão do servidor
	LastCleanupTime      time.Time `json:"last_cleanup_time"`      // Quando a última limpeza foi executada
	StaleNetworksRemoved int       `json:"stale_networks_removed"` // Número de salas obsoletas removidas
}

// NewStatsManager cria uma nova instância do gerenciador de estatísticas
func NewStatsManager(cfg Config) *StatsManager {
	logger.Debug("Initializing stats manager")
	return &StatsManager{
		stats: ServerStats{
			StartTime:            time.Now(),
			Version:              "1.0.0", // Versão hardcoded do servidor
			LastCleanupTime:      time.Time{},
			StaleNetworksRemoved: 0,
		},
		config: cfg,
	}
}

// UpdateStats atualiza as estatísticas com base nos dados atuais do servidor
func (sm *StatsManager) UpdateStats(activeConnections, activeNetworks int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Track if we hit a new peak for logging
	connectionPeak := false
	networksPeak := false

	// Atualiza estatísticas
	sm.stats.ActiveConnections = activeConnections
	sm.stats.ActiveNetworks = activeNetworks

	// Atualiza valores de pico se os contadores atuais forem mais altos
	if activeConnections > sm.stats.PeakConnections {
		sm.stats.PeakConnections = activeConnections
		connectionPeak = true
	}

	if activeNetworks > sm.stats.PeakNetworks {
		sm.stats.PeakNetworks = activeNetworks
		networksPeak = true
	}

	// Log new peaks when they happen
	if connectionPeak && sm.config.LogLevel == "debug" {
		logger.Debug("New peak connections reached",
			"connections", activeConnections)
	}

	if networksPeak && sm.config.LogLevel == "debug" {
		logger.Debug("New peak networks count reached",
			"networks", activeNetworks)
	}

	// Atualiza a string de tempo de atividade
	uptime := time.Since(sm.stats.StartTime)
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	sm.stats.Uptime = fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
}

// IncrementConnectionsTotal incrementa o contador de conexões totais
func (sm *StatsManager) IncrementConnectionsTotal() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.ConnectionsTotal++

	// Log connection milestones
	if sm.stats.ConnectionsTotal > 0 && sm.stats.ConnectionsTotal%100 == 0 {
		logger.Info("Connection milestone reached",
			"totalConnections", sm.stats.ConnectionsTotal)
	}
}

// IncrementMessagesProcessed incrementa o contador de mensagens processadas
func (sm *StatsManager) IncrementMessagesProcessed() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.MessagesProcessed++

	// Log message milestones
	if sm.stats.MessagesProcessed > 0 && sm.stats.MessagesProcessed%1000 == 0 {
		logger.Debug("Message milestone reached",
			"messagesProcessed", sm.stats.MessagesProcessed)
	}
}

// UpdateCleanupStats atualiza as estatísticas após uma operação de limpeza
func (sm *StatsManager) UpdateCleanupStats(numRemoved int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.LastCleanupTime = time.Now()
	sm.stats.StaleNetworksRemoved += numRemoved

	logger.Info("Network cleanup completed",
		"networksRemoved", numRemoved,
		"totalNetworksRemovedSinceStart", sm.stats.StaleNetworksRemoved,
		"timestamp", sm.stats.LastCleanupTime.Format(time.RFC3339))
}

// GetStats retorna uma cópia da estrutura de estatísticas atual
func (sm *StatsManager) GetStats() ServerStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.stats
}

// HandleStatsEndpoint é o manipulador HTTP para o endpoint /stats
func (sm *StatsManager) HandleStatsEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Log stats request
	logger.Debug("Stats endpoint requested",
		"remoteAddr", r.RemoteAddr,
		"userAgent", r.UserAgent())

	// Obtem estatísticas atualizadas
	stats := sm.GetStats()

	// Cria resposta com estatísticas atuais
	statsResponse := map[string]interface{}{
		"server_stats": stats,
		"config": map[string]interface{}{
			"max_clients_per_network": sm.config.MaxClientsPerNetwork,
			"network_expiry_days":     sm.config.NetworkExpiryDays,
			"cleanup_interval":        sm.config.CleanupInterval.String(),
			"allow_all_origins":       sm.config.AllowAllOrigins,
		},
	}

	// Converte para JSON e envia resposta
	jsonBytes, err := json.Marshal(statsResponse)
	if err != nil {
		logger.Error("Failed to generate statistics JSON", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to generate statistics"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
