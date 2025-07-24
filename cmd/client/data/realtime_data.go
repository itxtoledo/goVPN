package data

import (
	"log"
	"sync"

	"fyne.io/fyne/v2/data/binding"
	smodels "github.com/itxtoledo/govpn/libs/signaling/models"
)

// ConnectionState representa o estado da conexão
type ConnectionState int

const (
	// StateDisconnected representa o estado desconectado
	StateDisconnected ConnectionState = iota
	// StateConnecting representa o estado conectando
	StateConnecting
	// StateConnected representa o estado conectado
	StateConnected
)

// EventType representa o tipo de evento
type EventType string

const (
	// EventConnectionStateChanged é emitido quando o estado da conexão muda
	EventConnectionStateChanged EventType = "connection_state_changed"
	// EventNetworkJoined é emitido quando entra em uma sala
	EventNetworkJoined EventType = "network_joined"
	// EventNetworkLeft é emitido quando sai de uma sala
	EventNetworkLeft EventType = "network_left"
	// EventNetworkDisconnected é emitido quando desconecta de uma sala sem sair dela
	EventNetworkDisconnected EventType = "network_disconnected"
	EventNetworkDeleted      EventType = "network_deleted" // Add this constant for network deletion event
	EventSettingsChanged     EventType = "settings_changed"
	// EventError é emitido quando ocorre um erro
	EventError EventType = "error"
)

// ComputerInfo represents information about a computer in a network
type ComputerInfo = smodels.ComputerInfo

// Network represents a VPN network
type Network = smodels.ComputerNetworkInfo

// Event representa um evento da camada de dados
type Event struct {
	Type    EventType
	Message string
	Data    interface{}
}

// RealtimeDataLayer é a camada centralizada de dados em tempo real
type RealtimeDataLayer struct {
	// Dados de conexão
	ConnectionState binding.Int
	IsConnected     binding.Bool
	StatusMessage   binding.String

	// Dados de usuário
	ComputerName binding.String
	ComputerIP   binding.String

	// Dados de configuração
	ServerAddress binding.String
	Language      binding.String

	// Dados de rede
	ComputersCount   binding.Int
	NetworkLatency   binding.Float
	TransferredBytes binding.Float
	ReceivedBytes    binding.Float
	PublicKey        binding.String // Public key identifier

	// Dados de sala
	NetworkName binding.String
	Networks    binding.UntypedList // Lista de salas do usuário

	// Canal de eventos
	eventChan   chan Event
	subscribers []chan Event
	mu          sync.Mutex
}

// NewRealtimeDataLayer cria uma nova instância da camada de dados em tempo real
func NewRealtimeDataLayer() *RealtimeDataLayer {
	rdl := &RealtimeDataLayer{
		// Inicialização dos bindings
		ConnectionState:  binding.NewInt(),
		IsConnected:      binding.NewBool(),
		StatusMessage:    binding.NewString(),
		ComputerName:     binding.NewString(),
		ComputerIP:       binding.NewString(),
		ServerAddress:    binding.NewString(),
		Language:         binding.NewString(),
		ComputersCount:   binding.NewInt(),
		NetworkLatency:   binding.NewFloat(),
		TransferredBytes: binding.NewFloat(),
		ReceivedBytes:    binding.NewFloat(),
		PublicKey:        binding.NewString(),
		NetworkName:      binding.NewString(),
		Networks:         binding.NewUntypedList(),

		// Canal de eventos
		eventChan:   make(chan Event, 100),
		subscribers: make([]chan Event, 0),
	}

	// Iniciar o processamento de eventos
	go rdl.processEvents()

	return rdl
}

// InitDefaults inicializa os valores padrão
func (rdl *RealtimeDataLayer) InitDefaults() {
	// Valores padrão
	rdl.SetConnectionState(StateDisconnected)
	rdl.SetStatusMessage("Not connected")
	rdl.SetComputerName("Computer")
	rdl.SetComputerIP("0.0.0.0")
	rdl.SetServerAddress("ws://localhost:8080")
	rdl.SetLanguage("en")
	rdl.SetNetworkInfo("Not connected")
	rdl.UpdateNetworkStats(0, 0.0, 0.0, 0.0)
}

// SetConnectionState define o estado da conexão
func (rdl *RealtimeDataLayer) SetConnectionState(state ConnectionState) {
	rdl.ConnectionState.Set(int(state))
	rdl.IsConnected.Set(state == StateConnected)

	// Emitir evento
	rdl.EmitEvent(EventConnectionStateChanged, "", state)
}

// SetStatusMessage define a mensagem de status
func (rdl *RealtimeDataLayer) SetStatusMessage(message string) {
	rdl.StatusMessage.Set(message)
}

// SetComputerName define o nome de usuário
func (rdl *RealtimeDataLayer) SetComputerName(computername string) {
	rdl.ComputerName.Set(computername)
}

// SetComputerIP define o IP do usuário
func (rdl *RealtimeDataLayer) SetComputerIP(ip string) {
	rdl.ComputerIP.Set(ip)
}

// SetServerAddress define o endereço do servidor
func (rdl *RealtimeDataLayer) SetServerAddress(address string) {
	rdl.ServerAddress.Set(address)
}

// SetLanguage define o idioma da interface
func (rdl *RealtimeDataLayer) SetLanguage(lang string) {
	rdl.Language.Set(lang)
}

// UpdateNetworkStats atualiza as estatísticas de rede
func (rdl *RealtimeDataLayer) UpdateNetworkStats(computersCount int, latency, sent, received float64) {
	rdl.ComputersCount.Set(computersCount)
	rdl.NetworkLatency.Set(latency)
	rdl.TransferredBytes.Set(sent)
	rdl.ReceivedBytes.Set(received)
}

// SetNetworkInfo define as informações da sala
func (rdl *RealtimeDataLayer) SetNetworkInfo(name string) {
	rdl.NetworkName.Set(name)
}

// SetNetworks define a lista completa de salas
func (rdl *RealtimeDataLayer) SetNetworks(networks []Network) {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	log.Printf("SetNetworks: Setting %d networks", len(networks))

	// Convert []*Network to []interface{}
	var untypedNetworks []interface{}
	for _, network := range networks {
		untypedNetworks = append(untypedNetworks, network)
	}
	rdl.Networks.Set(untypedNetworks)
}

// AddNetwork adiciona uma nova sala à lista
func (rdl *RealtimeDataLayer) AddNetwork(network Network) {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	currentNetworks, _ := rdl.Networks.Get()
	for _, existing := range currentNetworks {
		if n, ok := existing.(*Network); ok && n.NetworkID == network.NetworkID {
			// Network already exists, do not add
			return
		}
	}
	rdl.Networks.Set(append(currentNetworks, network))
}

// RemoveNetwork remove uma sala da lista pelo ID
func (rdl *RealtimeDataLayer) RemoveNetwork(networkID string) {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	currentNetworks, _ := rdl.Networks.Get()
	var updatedNetworks []interface{}
	for _, r := range currentNetworks {
		if network, ok := r.(*Network); ok && network.NetworkID != networkID {
			updatedNetworks = append(updatedNetworks, network)
		}
	}
	rdl.Networks.Set(updatedNetworks)
}

// UpdateNetwork atualiza uma sala existente na lista
func (rdl *RealtimeDataLayer) UpdateNetwork(index int, network Network) {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	currentNetworks, _ := rdl.Networks.Get()
	if index >= 0 && index < len(currentNetworks) {
		currentNetworks[index] = network
		rdl.Networks.Set(currentNetworks)
	}
}

// GetNetworks retorna a lista atual de salas
func (rdl *RealtimeDataLayer) GetNetworks() []Network {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	currentNetworks, _ := rdl.Networks.Get()
	log.Printf("GetNetworks: Retrieved %d networks", len(currentNetworks))
	networks := make([]Network, len(currentNetworks))
	for i, r := range currentNetworks {
		if network, ok := r.(Network); ok {
			networks[i] = network
		} else {
			log.Printf("GetNetworks: Type assertion failed for network at index %d", i)
		}
	}
	return networks
}

// Subscribe inscreve um novo assinante para eventos
func (rdl *RealtimeDataLayer) Subscribe() chan Event {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	ch := make(chan Event, 10)
	rdl.subscribers = append(rdl.subscribers, ch)
	return ch
}

// Unsubscribe cancela a inscrição de um assinante
func (rdl *RealtimeDataLayer) Unsubscribe(ch chan Event) {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	for i, subscriber := range rdl.subscribers {
		if subscriber == ch {
			rdl.subscribers = append(rdl.subscribers[:i], rdl.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// EmitEvent emite um evento para todos os assinantes
func (rdl *RealtimeDataLayer) EmitEvent(eventType EventType, message string, data interface{}) {
	event := Event{
		Type:    eventType,
		Message: message,
		Data:    data,
	}

	// Enviar ao canal de processamento
	rdl.eventChan <- event
}

// processEvents processa eventos e os distribui para assinantes
func (rdl *RealtimeDataLayer) processEvents() {
	for event := range rdl.eventChan {
		rdl.mu.Lock()
		subscribers := make([]chan Event, len(rdl.subscribers))
		copy(subscribers, rdl.subscribers)
		rdl.mu.Unlock()

		// Distribuir para todos os assinantes
		for _, subscriber := range subscribers {
			// Envio não bloqueante
			select {
			case subscriber <- event:
				// Evento enviado com sucesso
			default:
				// Canal cheio, ignorar
			}
		}
	}
}
