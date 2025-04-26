package data

import (
	"sync"

	"fyne.io/fyne/v2/data/binding"
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
	// EventRoomJoined é emitido quando entra em uma sala
	EventRoomJoined EventType = "room_joined"
	// EventRoomLeft é emitido quando sai de uma sala
	EventRoomLeft        EventType = "room_left"
	EventRoomDeleted               = "room_deleted" // Add this constant for room deletion event
	EventSettingsChanged EventType = "settings_changed"
	// EventError é emitido quando ocorre um erro
	EventError EventType = "error"
)

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
	Username binding.String

	// Dados de configuração
	ServerAddress binding.String
	Language      binding.String

	// Dados de rede
	LocalIP          binding.String
	PeerCount        binding.Int
	NetworkLatency   binding.Float
	TransferredBytes binding.Float
	ReceivedBytes    binding.Float
	PublicKey        binding.String // Public key identifier

	// Dados de sala
	RoomName     binding.String
	RoomPassword binding.String

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
		Username:         binding.NewString(),
		ServerAddress:    binding.NewString(),
		Language:         binding.NewString(),
		LocalIP:          binding.NewString(),
		PeerCount:        binding.NewInt(),
		NetworkLatency:   binding.NewFloat(),
		TransferredBytes: binding.NewFloat(),
		ReceivedBytes:    binding.NewFloat(),
		PublicKey:        binding.NewString(),
		RoomName:         binding.NewString(),
		RoomPassword:     binding.NewString(),

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
	rdl.SetUsername("User")
	rdl.SetServerAddress("ws://localhost:8080")
	rdl.SetLanguage("en")
	rdl.SetLocalIP("YOUR IPV4")
	rdl.SetRoomInfo("Not connected", "")
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

// SetUsername define o nome de usuário
func (rdl *RealtimeDataLayer) SetUsername(username string) {
	rdl.Username.Set(username)
}

// SetServerAddress define o endereço do servidor
func (rdl *RealtimeDataLayer) SetServerAddress(address string) {
	rdl.ServerAddress.Set(address)
}

// SetLanguage define o idioma da interface
func (rdl *RealtimeDataLayer) SetLanguage(lang string) {
	rdl.Language.Set(lang)
}

// SetLocalIP define o IP local
func (rdl *RealtimeDataLayer) SetLocalIP(ip string) {
	rdl.LocalIP.Set(ip)
}

// UpdateNetworkStats atualiza as estatísticas de rede
func (rdl *RealtimeDataLayer) UpdateNetworkStats(peerCount int, latency, sent, received float64) {
	rdl.PeerCount.Set(peerCount)
	rdl.NetworkLatency.Set(latency)
	rdl.TransferredBytes.Set(sent)
	rdl.ReceivedBytes.Set(received)
}

// SetRoomInfo define as informações da sala
func (rdl *RealtimeDataLayer) SetRoomInfo(name, password string) {
	rdl.RoomName.Set(name)
	rdl.RoomPassword.Set(password)
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
