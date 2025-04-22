package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"

	"golang.org/x/crypto/pbkdf2"
	"crypto/sha256"
	"github.com/pion/webrtc/v3"
)

// VirtualNetwork gerencia a rede virtual entre os clientes
type VirtualNetwork struct {
	localIP        string
	interfaces     map[string]net.Interface
	connections    map[string]*webrtc.DataChannel
	roomID         string
	roomPassword   string
	encryptTraffic bool
	mu             sync.RWMutex
}

// NewVirtualNetwork cria uma nova rede virtual
func NewVirtualNetwork(roomID, password string) *VirtualNetwork {
	return &VirtualNetwork{
		localIP:        "10.0.0.1", // IP base, será incrementado para cada cliente
		interfaces:     make(map[string]net.Interface),
		connections:    make(map[string]*webrtc.DataChannel),
		roomID:         roomID,
		roomPassword:   password,
		encryptTraffic: true,
	}
}

// AddPeer adiciona um novo peer à rede virtual
func (v *VirtualNetwork) AddPeer(peerID string, dataChannel *webrtc.DataChannel) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Gera um endereço IP virtual para o novo peer
	newIP := fmt.Sprintf("10.0.0.%d", len(v.connections)+2) // +2 porque começamos em 10.0.0.1 e o primeiro cliente é 10.0.0.2

	// Configura a interface virtual
	iface := net.Interface{
		Name:  fmt.Sprintf("vpn%d", len(v.connections)),
		Flags: net.FlagUp | net.FlagPointToPoint,
	}

	v.interfaces[peerID] = iface
	v.connections[peerID] = dataChannel

	// Envia o IP atribuído ao peer
	ipAssignment := struct {
		Type    string `json:"type"`
		IP      string `json:"ip"`
		Netmask string `json:"netmask"`
	}{
		Type:    "IPAssignment",
		IP:      newIP,
		Netmask: "255.255.255.0",
	}

	data, err := json.Marshal(ipAssignment)
	if err != nil {
		return fmt.Errorf("erro ao serializar atribuição IP: %v", err)
	}

	// Se a VPN estiver configurada para criptografar o tráfego, criptografa o pacote
	if v.encryptTraffic {
		key := pbkdf2.Key([]byte(v.roomPassword), []byte(v.roomID), 4096, 32, sha256.New)
		data, err = encrypt(data, key)
		if err != nil {
			return fmt.Errorf("erro ao criptografar pacote: %v", err)
		}
	}

	// Envia o pacote pelo canal de dados WebRTC
	err = dataChannel.Send(data)
	if err != nil {
		return fmt.Errorf("erro ao enviar pacote: %v", err)
	}

	log.Printf("Peer %s adicionado à rede virtual com IP %s", peerID, newIP)
	return nil
}

// RemovePeer remove um peer da rede virtual
func (v *VirtualNetwork) RemovePeer(peerID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.interfaces, peerID)
	delete(v.connections, peerID)
	log.Printf("Peer %s removido da rede virtual", peerID)
}

// SendPacket envia um pacote para outro peer na rede
func (v *VirtualNetwork) SendPacket(destIP string, protocol string, port int, data []byte) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Encontra o peer com o IP de destino
	var targetPeerID string
	for peerID, iface := range v.interfaces {
		// Normalmente teríamos que consultar o IP da interface, mas estamos usando uma simplificação
		// onde o ID do peer é mapeado para um IP gerado de forma sequencial
		ifaceIP := fmt.Sprintf("10.0.0.%s", peerID[len(peerID)-1:])
		if ifaceIP == destIP {
			targetPeerID = peerID
			break
		}
	}

	if targetPeerID == "" {
		return fmt.Errorf("IP de destino não encontrado na rede virtual: %s", destIP)
	}

	// Prepara o pacote para envio
	packet := NetworkPacket{
		Source:      v.localIP,
		Destination: destIP,
		Protocol:    protocol,
		Port:        port,
		Data:        data,
		Encrypted:   v.encryptTraffic,
	}

	packetData, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("erro ao serializar pacote: %v", err)
	}

	// Se necessário, criptografa o pacote
	if v.encryptTraffic {
		key := pbkdf2.Key([]byte(v.roomPassword), []byte(v.roomID), 4096, 32, sha256.New)
		packetData, err = encrypt(packetData, key)
		if err != nil {
			return fmt.Errorf("erro ao criptografar pacote: %v", err)
		}
	}

	// Envia o pacote pelo canal de dados WebRTC
	dataChannel := v.connections[targetPeerID]
	if dataChannel == nil {
		return fmt.Errorf("canal de dados não encontrado para o peer %s", targetPeerID)
	}

	return dataChannel.Send(packetData)
}

// HandleIncomingPacket processa um pacote recebido
func (v *VirtualNetwork) HandleIncomingPacket(peerID string, data []byte) ([]byte, error) {
	// Se necessário, descriptografa o pacote
	if v.encryptTraffic {
		key := pbkdf2.Key([]byte(v.roomPassword), []byte(v.roomID), 4096, 32, sha256.New)
		var err error
		data, err = decrypt(data, key)
		if err != nil {
			return nil, fmt.Errorf("erro ao descriptografar pacote: %v", err)
		}
	}

	// Desserializa o pacote
	var packet NetworkPacket
	if err := json.Unmarshal(data, &packet); err != nil {
		return nil, fmt.Errorf("erro ao desserializar pacote: %v", err)
	}

	// Verifica se o pacote é para esta máquina
	if packet.Destination != v.localIP {
		// Repassa o pacote para o destino correto
		return nil, v.SendPacket(packet.Destination, packet.Protocol, packet.Port, packet.Data)
	}

	// O pacote é para esta máquina, retorna os dados
	return packet.Data, nil
}

// Inicializa uma interface de rede virtual (TUN/TAP) para tráfego real
// Esta é uma implementação simplificada - para um produto real você precisaria
// configurar um dispositivo TUN/TAP, o que requer privilégios elevados e
// é dependente do sistema operacional
func (v *VirtualNetwork) InitializeVirtualInterface() error {
	// Esta é uma implementação de exemplo para mostrar o conceito
	// Em uma implementação real, você usaria algo como:
	// - Linux: package github.com/songgao/water
	// - Windows: WinTUN ou OpenVPN TAP driver
	// - macOS: utun device ou implementação baseada em pf

	log.Println("Inicializando interface virtual (simulação)")
	return nil
}

// Captura pacotes da interface de rede real e os envia para a VPN
func (v *VirtualNetwork) CaptureAndSendPackets() {
	// Implementação simplificada
	log.Println("Iniciando captura de pacotes (simulação)")
}