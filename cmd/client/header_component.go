// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/header_component.go
package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// HeaderComponent representa o cabeçalho da aplicação
type HeaderComponent struct {
	UI            *UIManager
	PowerButton   *widget.Button
	IPInfoLabel   *widget.Label
	RoomNameLabel *widget.Label
}

// NewHeaderComponent cria uma nova instância do componente de cabeçalho
func NewHeaderComponent(ui *UIManager) *HeaderComponent {
	header := &HeaderComponent{
		UI: ui,
	}

	header.init()
	return header
}

// init inicializa os componentes do cabeçalho
func (h *HeaderComponent) init() {
	// Cria o botão de ligar/desligar
	h.PowerButton = widget.NewButtonWithIcon("", h.loadPowerButtonResource(), func() {
		h.handlePowerButtonClick()
	})
	h.PowerButton.Importance = widget.DangerImportance

	// Label para informações de IP
	h.IPInfoLabel = widget.NewLabel("YOUR IPV4")

	// Label para nome da sala
	h.RoomNameLabel = widget.NewLabel("Sala: Não conectado")
	h.RoomNameLabel.Alignment = fyne.TextAlignTrailing
}

// handlePowerButtonClick lida com o clique no botão de power
func (h *HeaderComponent) handlePowerButtonClick() {
	if h.UI.VPN.IsConnected {
		// Desconectar
		err := h.UI.VPN.NetworkManager.LeaveRoom()
		if err != nil {
			log.Printf("Erro ao desconectar: %v", err)
		}
		h.UI.VPN.IsConnected = false
		h.updatePowerButtonState()
		h.updateIPInfo()
		h.updateRoomName()
		h.UI.refreshNetworkList()
	} else {
		// Conectar
		// Certifique-se de que a janela de conexão está inicializada
		if h.UI.ConnectWindow == nil {
			h.UI.ConnectWindow = NewConnectWindow(h.UI)
		}
		h.UI.ConnectWindow.Show()
	}
}

// CreateHeaderContainer cria o container do cabeçalho
func (h *HeaderComponent) CreateHeaderContainer() *fyne.Container {
	// Definindo uma altura fixa para o header
	headerHeight := 60.0
	maxWidth := 300.0

	// Container para o botão power centralizado verticalmente
	powerContainer := container.New(layout.NewCenterLayout(), h.PowerButton)
	powerContainer.Resize(fyne.NewSize(40, float32(headerHeight)))

	// Container para a informação de IP centralizada verticalmente
	// Reduzindo largura para garantir que não ultrapasse o limite
	ipContainer := container.New(layout.NewCenterLayout(), h.IPInfoLabel)
	ipContainer.Resize(fyne.NewSize(180, float32(headerHeight)))

	// Configura o layout IP para garantir que o texto seja exibido corretamente
	h.IPInfoLabel.Wrapping = fyne.TextTruncate
	h.IPInfoLabel.Resize(fyne.NewSize(170, 30))

	// Container principal do header com layout horizontal e padding
	headerTop := container.New(
		layout.NewHBoxLayout(),
		powerContainer,
		layout.NewSpacer(),
		ipContainer,
	)
	headerTop.Resize(fyne.NewSize(float32(maxWidth), float32(headerHeight)))

	// Room name label com tamanho controlado
	h.RoomNameLabel.Wrapping = fyne.TextTruncate
	h.RoomNameLabel.Resize(fyne.NewSize(280, 20))

	roomNameContainer := container.NewHBox(
		layout.NewSpacer(),
		h.RoomNameLabel,
	)
	roomNameContainer.Resize(fyne.NewSize(float32(maxWidth), 20))

	// Container do cabeçalho completo com tamanho fixo
	header := container.NewVBox(
		headerTop,
		roomNameContainer,
	)
	header.Resize(fyne.NewSize(float32(maxWidth), float32(headerHeight+20)))

	return container.NewMax(header)
}

// updatePowerButtonState atualiza o estado visual do botão de ligar/desligar
func (h *HeaderComponent) updatePowerButtonState() {
	if h.UI.VPN.IsConnected {
		h.PowerButton.Importance = widget.HighImportance // Verde para conectado
	} else {
		h.PowerButton.Importance = widget.DangerImportance // Vermelho para desconectado
	}
	h.PowerButton.Refresh()
}

// updateIPInfo atualiza as informações de IP exibidas
func (h *HeaderComponent) updateIPInfo() {
	ipv4 := "YOUR IPV4"

	if h.UI.VPN.IsConnected && h.UI.VPN.NetworkManager.VirtualNetwork != nil {
		ipv4 = h.UI.VPN.NetworkManager.VirtualNetwork.GetLocalIP()
	}

	h.IPInfoLabel.SetText(ipv4)
	h.IPInfoLabel.Refresh()
}

// updateRoomName atualiza o nome da sala exibido
func (h *HeaderComponent) updateRoomName() {
	roomName := "Não conectado"

	if h.UI.VPN.IsConnected && h.UI.VPN.NetworkManager.RoomName != "" {
		roomName = h.UI.VPN.NetworkManager.RoomName
	}

	h.RoomNameLabel.SetText("Sala: " + roomName)
	h.RoomNameLabel.Refresh()
}

// loadPowerButtonResource carrega o ícone SVG do botão de energia
func (h *HeaderComponent) loadPowerButtonResource() fyne.Resource {
	// Carregar o ícone do arquivo SVG
	res, err := fyne.LoadResourceFromPath("power-button.svg")
	if err != nil {
		log.Printf("Erro ao carregar o ícone do botão de energia: %v", err)
		return theme.CancelIcon() // Retorna um ícone padrão em caso de falha
	}
	return res
}
