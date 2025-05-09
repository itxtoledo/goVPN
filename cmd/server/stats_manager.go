// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/server/stats_manager.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// StatsManager gerencia as estatísticas do servidor WebSocket
type StatsManager struct {
	stats  ServerStats
	mu     sync.RWMutex
	config Config
}

// ServerStats armazena várias métricas do servidor WebSocket
type ServerStats struct {
	StartTime         time.Time `json:"start_time"`          // Quando o servidor foi iniciado
	Uptime            string    `json:"uptime"`              // Tempo de atividade legível
	ConnectionsTotal  int       `json:"connections_total"`   // Total de conexões desde o início
	ActiveConnections int       `json:"active_connections"`  // Número atual de conexões ativas
	ActiveRooms       int       `json:"active_rooms"`        // Número de salas ativas
	PeakConnections   int       `json:"peak_connections"`    // Número máximo de conexões simultâneas
	PeakRooms         int       `json:"peak_rooms"`          // Número máximo de salas simultâneas
	MessagesProcessed int64     `json:"messages_processed"`  // Número de mensagens processadas
	Version           string    `json:"version"`             // Versão do servidor
	LastCleanupTime   time.Time `json:"last_cleanup_time"`   // Quando a última limpeza foi executada
	StaleRoomsRemoved int       `json:"stale_rooms_removed"` // Número de salas obsoletas removidas
}

// NewStatsManager cria uma nova instância do gerenciador de estatísticas
func NewStatsManager(cfg Config) *StatsManager {
	return &StatsManager{
		stats: ServerStats{
			StartTime:         time.Now(),
			Version:           "1.0.0", // Versão hardcoded do servidor
			LastCleanupTime:   time.Time{},
			StaleRoomsRemoved: 0,
		},
		config: cfg,
	}
}

// UpdateStats atualiza as estatísticas com base nos dados atuais do servidor
func (sm *StatsManager) UpdateStats(activeConnections, activeRooms int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Atualiza estatísticas
	sm.stats.ActiveConnections = activeConnections
	sm.stats.ActiveRooms = activeRooms

	// Atualiza valores de pico se os contadores atuais forem mais altos
	if activeConnections > sm.stats.PeakConnections {
		sm.stats.PeakConnections = activeConnections
	}

	if activeRooms > sm.stats.PeakRooms {
		sm.stats.PeakRooms = activeRooms
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
}

// IncrementMessagesProcessed incrementa o contador de mensagens processadas
func (sm *StatsManager) IncrementMessagesProcessed() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.MessagesProcessed++
}

// UpdateCleanupStats atualiza as estatísticas após uma operação de limpeza
func (sm *StatsManager) UpdateCleanupStats(numRemoved int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stats.LastCleanupTime = time.Now()
	sm.stats.StaleRoomsRemoved += numRemoved
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

	// Obtem estatísticas atualizadas
	stats := sm.GetStats()

	// Cria resposta com estatísticas atuais
	statsResponse := map[string]interface{}{
		"server_stats": stats,
		"config": map[string]interface{}{
			"max_clients_per_room": sm.config.MaxClientsPerRoom,
			"room_expiry_days":     sm.config.RoomExpiryDays,
			"cleanup_interval":     sm.config.CleanupInterval.String(),
			"allow_all_origins":    sm.config.AllowAllOrigins,
		},
	}

	// Converte para JSON e envia resposta
	jsonBytes, err := json.Marshal(statsResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to generate statistics"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
