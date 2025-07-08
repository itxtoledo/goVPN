package data

import (
	"sync"

	"fyne.io/fyne/v2/data/binding"
	"github.com/itxtoledo/govpn/cmd/client/storage"
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
	EventRoomLeft EventType = "room_left"
	// EventRoomDisconnected é emitido quando desconecta de uma sala sem sair dela
	EventRoomDisconnected EventType = "room_disconnected"
	EventRoomDeleted      EventType = "room_deleted" // Add this constant for room deletion event
	EventSettingsChanged  EventType = "settings_changed"
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
	UserIP   binding.String

	// Dados de configuração
	ServerAddress binding.String
	Language      binding.String

	// Dados de rede
	PeerCount        binding.Int
	NetworkLatency   binding.Float
	TransferredBytes binding.Float
	ReceivedBytes    binding.Float
	PublicKey        binding.String // Public key identifier

	// Dados de sala
	RoomName binding.String
	Rooms    binding.UntypedList // Lista de salas do usuário

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
		UserIP:           binding.NewString(),
		ServerAddress:    binding.NewString(),
		Language:         binding.NewString(),
		PeerCount:        binding.NewInt(),
		NetworkLatency:   binding.NewFloat(),
		TransferredBytes: binding.NewFloat(),
		ReceivedBytes:    binding.NewFloat(),
		PublicKey:        binding.NewString(),
		RoomName:         binding.NewString(),
		Rooms:            binding.NewUntypedList(),

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
	rdl.SetUserIP("0.0.0.0")
	rdl.SetServerAddress("ws://localhost:8080")
	rdl.SetLanguage("en")
	rdl.SetRoomInfo("Not connected")
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

// SetUserIP define o IP do usuário
func (rdl *RealtimeDataLayer) SetUserIP(ip string) {
	rdl.UserIP.Set(ip)
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
func (rdl *RealtimeDataLayer) UpdateNetworkStats(peerCount int, latency, sent, received float64) {
	rdl.PeerCount.Set(peerCount)
	rdl.NetworkLatency.Set(latency)
	rdl.TransferredBytes.Set(sent)
	rdl.ReceivedBytes.Set(received)
}

// SetRoomInfo define as informações da sala
func (rdl *RealtimeDataLayer) SetRoomInfo(name string) {
	rdl.RoomName.Set(name)
}

// SetRooms define a lista completa de salas
func (rdl *RealtimeDataLayer) SetRooms(rooms []*storage.Room) {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	// Convert []*storage.Room to []interface{}
	var untypedRooms []interface{}
	for _, room := range rooms {
		untypedRooms = append(untypedRooms, room)
	}
	rdl.Rooms.Set(untypedRooms)
}

// AddRoom adiciona uma nova sala à lista
func (rdl *RealtimeDataLayer) AddRoom(room *storage.Room) {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	currentRooms, _ := rdl.Rooms.Get()
	rdl.Rooms.Set(append(currentRooms, room))
}

// RemoveRoom remove uma sala da lista pelo ID
func (rdl *RealtimeDataLayer) RemoveRoom(roomID string) {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	currentRooms, _ := rdl.Rooms.Get()
	var updatedRooms []interface{}
	for _, r := range currentRooms {
		if room, ok := r.(*storage.Room); ok && room.ID != roomID {
			updatedRooms = append(updatedRooms, room)
		}
	}
	rdl.Rooms.Set(updatedRooms)
}

// UpdateRoom atualiza uma sala existente na lista
func (rdl *RealtimeDataLayer) UpdateRoom(index int, room *storage.Room) {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	currentRooms, _ := rdl.Rooms.Get()
	if index >= 0 && index < len(currentRooms) {
		currentRooms[index] = room
		rdl.Rooms.Set(currentRooms)
	}
}

// GetRooms retorna a lista atual de salas
func (rdl *RealtimeDataLayer) GetRooms() []*storage.Room {
	rdl.mu.Lock()
	defer rdl.mu.Unlock()

	currentRooms, _ := rdl.Rooms.Get()
	rooms := make([]*storage.Room, len(currentRooms))
	for i, r := range currentRooms {
		if room, ok := r.(*storage.Room); ok {
			rooms[i] = room
		}
	}
	return rooms
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
