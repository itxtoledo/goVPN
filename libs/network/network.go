package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"

	"crypto/sha256"

	"github.com/itxtoledo/govpn/libs/crypto_utils"
	"github.com/pion/webrtc/v3"
	"golang.org/x/crypto/pbkdf2"
)

// NetworkPacket representa um pacote de dados na rede virtual
type NetworkPacket struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Protocol    string `json:"protocol"`
	Port        int    `json:"port"`
	Data        []byte `json:"data"`
	Encrypted   bool   `json:"encrypted"`
}

// VirtualNetwork gerencia a rede virtual entre os clientes
type VirtualNetwork struct {
	localIP         string
	interfaces      map[string]net.Interface
	connections     map[string]*webrtc.DataChannel
	networkID       string
	networkPassword string
	encryptTraffic  bool
	mu              sync.RWMutex
}

// NewVirtualNetwork cria uma nova rede virtual
func NewVirtualNetwork(networkID, password string) *VirtualNetwork {
	return &VirtualNetwork{
		localIP:         "10.0.0.1", // IP base, será incrementado para cada cliente
		interfaces:      make(map[string]net.Interface),
		connections:     make(map[string]*webrtc.DataChannel),
		networkID:       networkID,
		networkPassword: password,
		encryptTraffic:  true,
	}
}

// GetLocalIP retorna o endereço IP local da rede virtual
func (v *VirtualNetwork) GetLocalIP() string {
	return v.localIP
}

// AddComputer adiciona um novo computer à rede virtual
func (v *VirtualNetwork) AddComputer(computerID string, dataChannel *webrtc.DataChannel) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Gera um endereço IP virtual para o novo computer
	newIP := fmt.Sprintf("10.0.0.%d", len(v.connections)+2) // +2 porque começamos em 10.0.0.1 e o primeiro cliente é 10.0.0.2

	// Configura a interface virtual
	iface := net.Interface{
		Name:  fmt.Sprintf("vpn%d", len(v.connections)),
		Flags: net.FlagUp | net.FlagPointToPoint,
	}

	v.interfaces[computerID] = iface
	v.connections[computerID] = dataChannel

	// Envia o IP atribuído ao computer
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
		key := pbkdf2.Key([]byte(v.networkPassword), []byte(v.networkID), 4096, 32, sha256.New)
		data, err = crypto_utils.Encrypt(data, key)
		if err != nil {
			return fmt.Errorf("erro ao criptografar pacote: %v", err)
		}
	}

	// Envia o pacote pelo canal de dados WebRTC
	err = dataChannel.Send(data)
	if err != nil {
		return fmt.Errorf("erro ao enviar pacote: %v", err)
	}

	log.Printf("Computer %s adicionado à rede virtual com IP %s", computerID, newIP)
	return nil
}

// RemoveComputer remove um computer da rede virtual
func (v *VirtualNetwork) RemoveComputer(computerID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.interfaces, computerID)
	delete(v.connections, computerID)
	log.Printf("Computer %s removido da rede virtual", computerID)
}

// SendPacket envia um pacote para outro computer na rede
func (v *VirtualNetwork) SendPacket(destIP string, protocol string, port int, data []byte) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Encontra o computer com o IP de destino
	var targetComputerID string
	for computerID := range v.interfaces {
		// Normalmente teríamos que consultar o IP da interface, mas estamos usando uma simplificação
		// onde o ID da máquina é mapeado para um IP gerado de forma sequencial
		ifaceIP := fmt.Sprintf("10.0.0.%s", computerID[len(computerID)-1:])
		if ifaceIP == destIP {
			targetComputerID = computerID
			break
		}
	}

	if targetComputerID == "" {
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
		key := pbkdf2.Key([]byte(v.networkPassword), []byte(v.networkID), 4096, 32, sha256.New)
		packetData, err = crypto_utils.Encrypt(packetData, key)
		if err != nil {
			return fmt.Errorf("erro ao criptografar pacote: %v", err)
		}
	}

	// Envia o pacote pelo canal de dados WebRTC
	dataChannel := v.connections[targetComputerID]
	if dataChannel == nil {
		return fmt.Errorf("canal de dados não encontrado para a máquina %s", targetComputerID)
	}

	return dataChannel.Send(packetData)
}

// HandleIncomingPacket processa um pacote recebido
func (v *VirtualNetwork) HandleIncomingPacket(computerID string, data []byte) ([]byte, error) {
	// Se necessário, descriptografa o pacote
	if v.encryptTraffic {
		key := pbkdf2.Key([]byte(v.networkPassword), []byte(v.networkID), 4096, 32, sha256.New)
		var err error
		data, err = crypto_utils.Decrypt(data, key)
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

func (v *VirtualNetwork) GetNetworkPassword() string {
	return v.networkPassword
}
