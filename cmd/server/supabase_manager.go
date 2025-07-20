package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/itxtoledo/govpn/cmd/server/logger"
	"github.com/itxtoledo/govpn/libs/models"
	"github.com/supabase-community/supabase-go"
)

// SupabaseManager handles all Supabase database operations for the server
type SupabaseManager struct {
	client        *supabase.Client
	networksTable string
	logLevel      string
}

// NewSupabaseManager creates a new instance of the Supabase manager
func NewSupabaseManager(supabaseURL, supabaseKey, networksTable, logLevel string) (*SupabaseManager, error) {
	client, err := supabase.NewClient(supabaseURL, supabaseKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Supabase client: %w", err)
	}

	return &SupabaseManager{
		client:        client,
		networksTable: networksTable,
		logLevel:      logLevel,
	}, nil
}

// CreateNetwork inserts a new network into the Supabase database
func (sm *SupabaseManager) CreateNetwork(network ServerNetwork) error {
	networkData := map[string]interface{}{
		"id":          network.ID,
		"name":        network.Name,
		"pin":         network.PIN,
		"public_key":  network.PublicKeyB64,
		"created_at":  network.CreatedAt.Format(time.RFC3339),
		"last_active": network.LastActive.Format(time.RFC3339),
	}

	if sm.logLevel == "debug" {
		logger.Debug("Creating network in Supabase", "networkID", network.ID, "networkName", network.Name)
	}

	_, _, err := sm.client.From(sm.networksTable).Insert(networkData, false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to create network in Supabase: %w", err)
	}

	return nil
}

// GetNetwork fetches a network from the Supabase database by its ID
func (sm *SupabaseManager) GetNetwork(networkID string) (ServerNetwork, error) {
	var networks []SupabaseNetwork
	data, _, err := sm.client.From(sm.networksTable).Select("*", "", false).Eq("id", networkID).Execute()
	if err != nil {
		return ServerNetwork{}, fmt.Errorf("failed to fetch network from Supabase: %w", err)
	}

	if err := json.Unmarshal(data, &networks); err != nil {
		return ServerNetwork{}, fmt.Errorf("failed to parse network data: %w", err)
	}

	if len(networks) == 0 {
		return ServerNetwork{}, fmt.Errorf("network not found: %s", networkID)
	}

	dbNetwork := networks[0]

	// Create a ServerNetwork from the SupabaseNetwork data
	return ServerNetwork{
		Network: models.Network{
			ID:   dbNetwork.ID,
			Name: dbNetwork.Name,
			PIN:  dbNetwork.PIN,
		},
		PublicKeyB64: dbNetwork.PublicKey,
		CreatedAt:    dbNetwork.CreatedAt,
		LastActive:   dbNetwork.LastActive,
	}, nil
}

// UpdateNetworkActivity updates the last_active timestamp for a network
func (sm *SupabaseManager) UpdateNetworkActivity(networkID string) error {
	now := time.Now().Format(time.RFC3339)
	updateData := map[string]interface{}{
		"last_active": now,
	}

	if sm.logLevel == "debug" {
		logger.Debug("Updating last_active for network", "networkID", networkID, "timestamp", now)
	}

	_, _, err := sm.client.From(sm.networksTable).Update(updateData, "", "").Eq("id", networkID).Execute()
	if err != nil {
		return fmt.Errorf("failed to update network activity: %w", err)
	}

	return nil
}

// UpdateNetworkName updates the name of a network
func (sm *SupabaseManager) UpdateNetworkName(networkID, newName string) error {
	updateData := map[string]interface{}{
		"name":        newName,
		"last_active": time.Now().Format(time.RFC3339),
	}

	if sm.logLevel == "debug" {
		logger.Debug("Updating name for network", "networkID", networkID, "newName", newName)
	}

	_, _, err := sm.client.From(sm.networksTable).Update(updateData, "", "").Eq("id", networkID).Execute()
	if err != nil {
		return fmt.Errorf("failed to update network name: %w", err)
	}

	return nil
}

// DeleteNetwork removes a network from the Supabase database
func (sm *SupabaseManager) DeleteNetwork(networkID string) error {
	if sm.logLevel == "debug" {
		logger.Debug("Deleting network from Supabase", "networkID", networkID)
	}

	_, _, err := sm.client.From(sm.networksTable).Delete("", "").Eq("id", networkID).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	return nil
}

// GetNetworkByPublicKey fetches a network by the owner's public key
func (sm *SupabaseManager) GetNetworkByPublicKey(publicKey string) (ServerNetwork, error) {
	var networks []SupabaseNetwork
	data, _, err := sm.client.From(sm.networksTable).Select("*", "", false).Eq("public_key", publicKey).Execute()
	if err != nil {
		return ServerNetwork{}, fmt.Errorf("failed to fetch network by public key: %w", err)
	}

	if err := json.Unmarshal(data, &networks); err != nil {
		return ServerNetwork{}, fmt.Errorf("failed to parse network data: %w", err)
	}

	if len(networks) == 0 {
		return ServerNetwork{}, fmt.Errorf("no network found for public key")
	}

	dbNetwork := networks[0]

	return ServerNetwork{
		Network: models.Network{
			ID:   dbNetwork.ID,
			Name: dbNetwork.Name,
			PIN:  dbNetwork.PIN,
		},
		PublicKeyB64: dbNetwork.PublicKey,
		CreatedAt:    dbNetwork.CreatedAt,
		LastActive:   dbNetwork.LastActive,
	}, nil
}

// GetStaleNetworks fetches networks that have not been active for a specified period
func (sm *SupabaseManager) GetStaleNetworks(expiryDays int) ([]SupabaseNetwork, error) {
	expiryDuration := time.Hour * 24 * time.Duration(expiryDays)
	cutoffTime := time.Now().Add(-expiryDuration)
	cutoffTimeStr := cutoffTime.Format(time.RFC3339)

	if sm.logLevel == "debug" {
		logger.Debug("Fetching stale networks", "cutoffTime", cutoffTimeStr, "expiryDays", expiryDays)
	}

	var staleNetworks []SupabaseNetwork
	data, _, err := sm.client.From(sm.networksTable).Select("*", "", false).Lt("last_active", cutoffTimeStr).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stale networks: %w", err)
	}

	if err := json.Unmarshal(data, &staleNetworks); err != nil {
		return nil, fmt.Errorf("failed to parse stale networks data: %w", err)
	}

	return staleNetworks, nil
}

// NetworkExists checks if a network exists with the given ID
func (sm *SupabaseManager) NetworkExists(networkID string) (bool, error) {
	var networks []map[string]interface{}
	data, _, err := sm.client.From(sm.networksTable).Select("id", "", false).Eq("id", networkID).Execute()
	if err != nil {
		return false, fmt.Errorf("failed to check if network exists: %w", err)
	}

	if err := json.Unmarshal(data, &networks); err != nil {
		return false, fmt.Errorf("failed to parse network data: %w", err)
	}

	return len(networks) > 0, nil
}

// PublicKeyHasNetwork checks if a public key already has an associated network
func (sm *SupabaseManager) PublicKeyHasNetwork(publicKey string) (bool, string, error) {
	var networks []map[string]interface{}
	data, _, err := sm.client.From(sm.networksTable).Select("id", "", false).Eq("public_key", publicKey).Execute()
	if err != nil {
		return false, "", fmt.Errorf("failed to check if public key has network: %w", err)
	}

	if err := json.Unmarshal(data, &networks); err != nil {
		return false, "", fmt.Errorf("failed to parse network data: %w", err)
	}

	if len(networks) > 0 {
		return true, networks[0]["id"].(string), nil
	}
	return false, "", nil
}

// ComputerNetwork represents a record in the computer_networks table linking computers to networks
type ComputerNetwork struct {
	ID            int       `json:"id"`
	NetworkID     string    `json:"network_id"`
	PublicKey     string    `json:"public_key"`
	ComputerName  string    `json:"computername"`
	JoinedAt      time.Time `json:"joined_at"`
	LastConnected time.Time `json:"last_connected"`
	PeerIP        string    `json:"peer_ip"`
}

// AddComputerToNetwork adds a computer to a network in the computer_networks table
func (sm *SupabaseManager) AddComputerToNetwork(networkID, publicKey, computerName, peerIP string) error {
	computerNetworkData := map[string]interface{}{
		"network_id":     networkID,
		"public_key":     publicKey,
		"computername":   computerName,
		"joined_at":      time.Now().Format(time.RFC3339),
		"last_connected": time.Now().Format(time.RFC3339),
		"peer_ip":        peerIP,
	}

	if sm.logLevel == "debug" {
		logger.Debug("Adding computer to network", "networkID", networkID, "publicKey", publicKey)
	}

	_, _, err := sm.client.From("computer_networks").Insert(computerNetworkData, false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to add computer to network in Supabase: %w", err)
	}

	return nil
}

// UpdateComputerNetworkConnection updates the last_connected timestamp for a computer in a network
func (sm *SupabaseManager) UpdateComputerNetworkConnection(networkID, publicKey string) error {
	updateData := map[string]interface{}{
		"last_connected": time.Now().Format(time.RFC3339),
	}

	if sm.logLevel == "debug" {
		logger.Debug("Updating computer network connection", "networkID", networkID, "publicKey", publicKey)
	}

	_, _, err := sm.client.From("computer_networks").Update(updateData, "", "").Eq("network_id", networkID).Eq("public_key", publicKey).Execute()
	if err != nil {
		return fmt.Errorf("failed to update computer network connection: %w", err)
	}

	return nil
}

// RemoveComputerFromNetwork removes a computer from a network
func (sm *SupabaseManager) RemoveComputerFromNetwork(networkID, publicKey string) error {
	if sm.logLevel == "debug" {
		logger.Debug("Removing computer from network", "networkID", networkID, "publicKey", publicKey)
	}

	_, _, err := sm.client.From("computer_networks").Delete("", "").Eq("network_id", networkID).Eq("public_key", publicKey).Execute()
	if err != nil {
		return fmt.Errorf("failed to remove computer from network: %w", err)
	}

	return nil
}

// GetNetworkComputers gets all computers for a specific network
func (sm *SupabaseManager) GetNetworkComputers(networkID string) ([]ComputerNetwork, error) {
	var computerNetworks []ComputerNetwork
	data, _, err := sm.client.From("computer_networks").Select("*", "", false).Eq("network_id", networkID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get network computers: %w", err)
	}

	if err := json.Unmarshal(data, &computerNetworks); err != nil {
		return nil, fmt.Errorf("failed to parse computer network data: %w", err)
	}

	return computerNetworks, nil
}

// GetUsedIPsForNetwork fetches all used IPs for a specific network
func (sm *SupabaseManager) GetUsedIPsForNetwork(networkID string) ([]string, error) {
	var computerNetworks []struct {
		PeerIP string `json:"peer_ip"`
	}
	data, _, err := sm.client.From("computer_networks").Select("peer_ip", "", false).Eq("network_id", networkID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get used IPs: %w", err)
	}

	if err := json.Unmarshal(data, &computerNetworks); err != nil {
		return nil, fmt.Errorf("failed to parse used IPs data: %w", err)
	}

	ips := make([]string, 0, len(computerNetworks))
	for _, cn := range computerNetworks {
		if cn.PeerIP != "" {
			ips = append(ips, cn.PeerIP)
		}
	}

	return ips, nil
}

// GetComputerNetworks gets all networks a computer has joined
func (sm *SupabaseManager) GetComputerNetworks(publicKey string) ([]ComputerNetwork, error) {
	var computerNetworks []ComputerNetwork
	data, _, err := sm.client.From("computer_networks").Select("*", "", false).Eq("public_key", publicKey).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get computer networks: %w", err)
	}

	if err := json.Unmarshal(data, &computerNetworks); err != nil {
		return nil, fmt.Errorf("failed to parse computer network data: %w", err)
	}

	return computerNetworks, nil
}

func (sm *SupabaseManager) GetComputerInNetwork(networkID, publicKey string) (ComputerNetwork, error) {
	var computerNetworks []ComputerNetwork
	data, _, err := sm.client.From("computer_networks").Select("*", "", false).Eq("network_id", networkID).Eq("public_key", publicKey).Execute()
	if err != nil {
		return ComputerNetwork{}, fmt.Errorf("failed to get computer in network: %w", err)
	}

	if err := json.Unmarshal(data, &computerNetworks); err != nil {
		return ComputerNetwork{}, fmt.Errorf("failed to parse computer network data: %w", err)
	}

	if len(computerNetworks) == 0 {
		return ComputerNetwork{}, fmt.Errorf("computer not found in network")
	}

	return computerNetworks[0], nil
}

// IsComputerInNetwork checks if a computer is already in a network
func (sm *SupabaseManager) IsComputerInNetwork(networkID, publicKey string) (bool, error) {
	var computerNetworks []ComputerNetwork
	data, _, err := sm.client.From("computer_networks").Select("id", "", false).Eq("network_id", networkID).Eq("public_key", publicKey).Execute()
	if err != nil {
		return false, fmt.Errorf("failed to check if computer is in network: %w", err)
	}

	if err := json.Unmarshal(data, &computerNetworks); err != nil {
		return false, fmt.Errorf("failed to parse computer network data: %w", err)
	}

	return len(computerNetworks) > 0, nil
}
