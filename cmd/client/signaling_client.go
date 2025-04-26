package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/itxtoledo/govpn/libs/models"
)

// SignalingClient representa uma conexão com o servidor de sinalização
type SignalingClient struct {
	UI             *UIManager
	VPNClient      *VPNClient
	Conn           *websocket.Conn
	ServerAddress  string
	Connected      bool
	LastHeartbeat  time.Time
	MessageHandler func(messageType int, message []byte) error
}

// NewSignalingClient cria uma nova instância do servidor de sinalização
func NewSignalingClient(ui *UIManager) *SignalingClient {
	return &SignalingClient{
		UI:            ui,
		Connected:     false,
		LastHeartbeat: time.Now(),
	}
}

// SetVPNClient sets the reference to the VPNClient for key access
func (s *SignalingClient) SetVPNClient(client *VPNClient) {
	s.VPNClient = client
}

// Connect conecta ao servidor de sinalização
func (s *SignalingClient) Connect(serverAddress string) error {
	if s.Connected {
		// Já está conectado
		return nil
	}

	s.ServerAddress = serverAddress

	// Criar URL para conexão WebSocket
	u, err := url.Parse(serverAddress)
	if err != nil {
		log.Printf("Error parsing server address: %v", err)
		return err
	}

	log.Printf("Connecting to WebSocket server at %s", u.String())

	// Estabelecer conexão real com o servidor WebSocket
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("Error connecting to WebSocket server: %v", err)
		return err
	}
	s.Conn = conn

	// Configurar handler para mensagens recebidas
	go s.listenForMessages()

	// Marcar como conectado
	s.Connected = true
	s.LastHeartbeat = time.Now()

	return nil
}

// Disconnect desconecta do servidor de sinalização
func (s *SignalingClient) Disconnect() error {
	if !s.Connected {
		// Já está desconectado
		return nil
	}

	// Fechar a conexão se existir
	if s.Conn != nil {
		err := s.Conn.Close()
		if err != nil {
			return err
		}
		s.Conn = nil
	}

	// Marcar como desconectado
	s.Connected = false

	return nil
}

// SendHeartbeat envia um heartbeat para o servidor
func (s *SignalingClient) SendHeartbeat() error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	// Os heartbeats não são mais usados nesta versão do protocolo
	// Todos os timestamps são controlados pelo servidor

	return nil
}

// sendPackagedMessage empacota e envia mensagem para o backend
// Cria BaseRequest com a chave pública do cliente,
// gera ID da mensagem, empacota na struct SignalingMessage e envia via WebSocket
func (s *SignalingClient) sendPackagedMessage(msgType models.MessageType, payload interface{}) error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	// Gerar ID da mensagem
	messageID, err := models.GenerateMessageID()
	if err != nil {
		return fmt.Errorf("error generating message ID: %v", err)
	}

	// Extensão de BaseRequest - adicionar chave pública se disponível
	var publicKey string
	if s.VPNClient != nil {
		publicKey = s.VPNClient.PublicKeyStr
	}

	// Add public key to the payload if it's a map
	if payloadMap, ok := payload.(map[string]interface{}); ok {
		payloadMap["PublicKey"] = publicKey
		payload = payloadMap
	}

	// Serializar payload para JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error serializing payload: %v", err)
	}

	// Criar SignalingMessage
	message := models.SignalingMessage{
		ID:      messageID,
		Type:    msgType,
		Payload: payloadBytes,
	}

	// Enviar a mensagem para o servidor
	log.Printf("Sending message of type %s with ID %s", msgType, messageID)
	err = s.Conn.WriteJSON(message)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	return nil
}

// CreateRoom cria uma nova sala no servidor
func (s *SignalingClient) CreateRoom(name string, description string, password string) error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	log.Printf("Creating room: %s", name)

	// Criar payload para a requisição
	payload := &models.CreateRoomRequest{
		BaseRequest: models.BaseRequest{},
		RoomName:    name,
		Password:    password,
	}

	// Enviar solicitação de criação de sala usando a função de empacotamento
	return s.sendPackagedMessage(models.TypeCreateRoom, payload)
}

// JoinRoom entra em uma sala
func (s *SignalingClient) JoinRoom(roomID string, password string) error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	log.Printf("Joining room: %s", roomID)

	// Obter o nome de usuário das configurações
	username := s.UI.ConfigManager.GetConfig().Username

	// Criar payload para join room
	payload := &models.JoinRoomRequest{
		BaseRequest: models.BaseRequest{},
		RoomID:      roomID,
		Password:    password,
		Username:    username,
	}

	// Enviar solicitação para entrar na sala usando a função de empacotamento
	return s.sendPackagedMessage(models.TypeJoinRoom, payload)
}

// LeaveRoom sai de uma sala
func (s *SignalingClient) LeaveRoom(roomID string) error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	log.Printf("Leaving room: %s", roomID)

	// Criar payload para leave room
	payload := &models.LeaveRoomRequest{
		BaseRequest:  models.BaseRequest{},
		RoomID:       roomID,
		PreserveRoom: true, // Por padrão, preserva a sala quando um cliente sai
	}

	// Enviar solicitação para sair da sala usando a função de empacotamento
	return s.sendPackagedMessage(models.TypeLeaveRoom, payload)
}

// GetRoomList obtém a lista de salas do servidor
func (s *SignalingClient) GetRoomList() (string, error) {
	if !s.Connected || s.Conn == nil {
		return "", errors.New("not connected to server")
	}

	// Esta funcionalidade não está implementada no servidor atual
	// A lista de salas não está mais disponível por API
	return "[]", nil
}

// RenameRoom renomeia uma sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) RenameRoom(roomID string, newName string) error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	log.Printf("Renaming room %s to %s", roomID, newName)

	// Criar payload para rename room
	payload := &models.RenameRequest{
		BaseRequest: models.BaseRequest{},
		RoomID:      roomID,
		RoomName:    newName,
	}

	// Enviar solicitação para renomear a sala usando a função de empacotamento
	return s.sendPackagedMessage(models.TypeRename, payload)
}

// KickUser expulsa um usuário da sala (apenas o proprietário pode fazer isso)
func (s *SignalingClient) KickUser(roomID string, targetID string) error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	log.Printf("Kicking user %s from room %s", targetID, roomID)

	// Criar payload para kick user
	payload := &models.KickRequest{
		BaseRequest: models.BaseRequest{},
		RoomID:      roomID,
		TargetID:    targetID,
	}

	// Enviar solicitação para expulsar o usuário usando a função de empacotamento
	return s.sendPackagedMessage(models.TypeKick, payload)
}

// SendMessage envia uma mensagem para o servidor
func (s *SignalingClient) SendMessage(messageType string, payload map[string]interface{}) error {
	if !s.Connected || s.Conn == nil {
		return errors.New("not connected to server")
	}

	// Converter o messageType para o tipo apropriado
	msgType := models.MessageType(messageType)

	// Enviar a mensagem usando a função de empacotamento
	return s.sendPackagedMessage(msgType, payload)
}

// SetMessageHandler define o handler de mensagens
func (s *SignalingClient) SetMessageHandler(handler func(messageType int, message []byte) error) {
	s.MessageHandler = handler
}

// IsConnected retorna se está conectado ao servidor
func (s *SignalingClient) IsConnected() bool {
	return s.Connected
}

// listenForMessages recebe e processa mensagens vindas do servidor
func (s *SignalingClient) listenForMessages() {
	if s.Conn == nil {
		log.Println("Cannot listen for messages: no websocket connection")
		return
	}

	for {
		if !s.Connected || s.Conn == nil {
			log.Println("Connection closed, stopping message listener")
			return
		}

		msgType, message, err := s.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			s.Connected = false
			s.Conn = nil
			return
		}

		if s.MessageHandler != nil {
			if err := s.MessageHandler(msgType, message); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		} else {
			log.Printf("Received message but no handler is configured")
		}
	}
}
