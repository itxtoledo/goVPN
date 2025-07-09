// filepath: /Computers/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/header_component.go
package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/icon"
)

// HeaderComponent representa o componente de cabeçalho da aplicação
type HeaderComponent struct {
	UI                  *UIManager
	PowerButton         *widget.Button
	ComputerIPLabel     *widget.Label
	ComputerNameLabel   *widget.Label
	NetworkLabel        *widget.Label
	defaultWebsocketURL string
}

// NewHeaderComponent cria uma nova instância do componente de cabeçalho
func NewHeaderComponent(ui *UIManager, defaultWebsocketURL string) *HeaderComponent {
	hc := &HeaderComponent{
		UI: ui,
	}

	// Criar componentes de UI
	// Use theme.MediaPlayIcon instead of a custom icon to avoid potential nil issues
	hc.PowerButton = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		hc.toggleConnection()
	})
	hc.PowerButton.Importance = widget.HighImportance // Make power button more prominent

	// Criar label para IP do usuário
	hc.ComputerIPLabel = widget.NewLabelWithData(hc.UI.RealtimeData.ComputerIP)
	hc.ComputerIPLabel.TextStyle = fyne.TextStyle{Monospace: true} // Use monospace for IP

	// Criar labels com dados em tempo real vinculados
	hc.ComputerNameLabel = widget.NewLabelWithData(hc.UI.RealtimeData.ComputerName)
	hc.ComputerNameLabel.TextStyle = fyne.TextStyle{Bold: true} // Make computername more prominent

	hc.NetworkLabel = widget.NewLabelWithData(hc.UI.RealtimeData.NetworkName)

	// Configure listeners para atualização automática
	hc.configureListeners()

	return hc
}

// configureListeners configura os listeners para os bindings
func (hc *HeaderComponent) configureListeners() {
	// Listener para o estado da conexão
	hc.UI.RealtimeData.ConnectionState.AddListener(binding.NewDataListener(func() {
		hc.updateBackendStatus()
		hc.updatePowerButtonState()
	}))

	// Listener para o nome da sala
	hc.UI.RealtimeData.NetworkName.AddListener(binding.NewDataListener(func() {
		hc.updateNetworkName()
	}))

	// Listener para o nome de usuário
	hc.UI.RealtimeData.ComputerName.AddListener(binding.NewDataListener(func() {
		hc.updateComputerName()
	}))

	// Listener para o IP do usuário
	hc.UI.RealtimeData.ComputerIP.AddListener(binding.NewDataListener(func() {
		hc.updateComputerIP()
	}))
}

// CreateHeaderContainer cria o container do cabeçalho
func (hc *HeaderComponent) CreateHeaderContainer() *fyne.Container {

	// Container para informações do usuário (IP e nome) - layout compacto
	computerInfoContainer := container.NewVBox(
		hc.ComputerIPLabel,
		hc.ComputerNameLabel,
	)

	// Container para o power button com tamanho fixo para garantir proporção 1:1
	powerButtonContainer := container.NewWithoutLayout(hc.PowerButton)
	powerButtonContainer.Resize(fyne.NewSize(44, 44))
	hc.PowerButton.Move(fyne.NewPos(0, 0))
	hc.PowerButton.Resize(fyne.NewSize(44, 44))

	// Container superior usando HBox com duas colunas
	topContainer := container.NewHBox(
		powerButtonContainer,  // Primeira coluna: power button
		computerInfoContainer, // Segunda coluna: informações do usuário (IP e nome)
	)

	// Container principal
	headerContainer := container.NewVBox(
		topContainer,
		widget.NewSeparator(),
	)

	return headerContainer
}

// toggleConnection alterna o estado da conexão
func (hc *HeaderComponent) toggleConnection() {
	state, _ := hc.UI.RealtimeData.ConnectionState.Get()
	connectionState := data.ConnectionState(state)

	if connectionState == data.StateDisconnected {
		// Conectar
		go func() {
			log.Println("Connecting to VPN network...")
			hc.UI.VPN.Run(hc.defaultWebsocketURL, hc.UI.RealtimeData, hc.UI.refreshNetworkList, hc.UI.refreshUI)
		}()
	} else {
		// Desconectar
		go func() {
			log.Println("Disconnecting from VPN network...")
			if hc.UI.VPN.NetworkManager != nil {
				err := hc.UI.VPN.NetworkManager.Disconnect()
				if err != nil {
					log.Printf("Error disconnecting: %v", err)
				}
			}
		}()
	}
}

// updatePowerButtonState atualiza o estado do botão de energia
func (hc *HeaderComponent) updatePowerButtonState() {
	state, _ := hc.UI.RealtimeData.ConnectionState.Get()
	connectionState := data.ConnectionState(state)

	// Atualizar ícone do botão
	hc.PowerButton.SetIcon(icon.Power)
	if connectionState == data.StateDisconnected {
		hc.PowerButton.Importance = widget.HighImportance
	} else {
		hc.PowerButton.Importance = widget.DangerImportance
	}

	// Atualizar o botão
	hc.PowerButton.Refresh()
}

// updateComputerName atualiza o nome de usuário
func (hc *HeaderComponent) updateComputerName() {
	// O nome de usuário já está vinculado diretamente via binding
	// Esta função é mantida para compatibilidade e possíveis futuras extensões
}

// updateComputerIP atualiza o IP do usuário
func (hc *HeaderComponent) updateComputerIP() {
	// O IP do usuário já está vinculado diretamente via binding
	// Esta função é mantida para compatibilidade e possíveis futuras extensões
}

// updateNetworkName atualiza o nome da sala
func (hc *HeaderComponent) updateNetworkName() {
	// O nome da sala já está vinculado diretamente via binding
	// Esta função é mantida para compatibilidade e possíveis futuras extensões
}

// updateBackendStatus atualiza o status do backend
func (hc *HeaderComponent) updateBackendStatus() {
	// state, _ := hc.UI.RealtimeData.ConnectionState.Get()
	// connectionState := data.ConnectionState(state)

	// switch connectionState {
	// case data.StateDisconnected:
	// 	hc.BackendActivity.Stop() // Para a animação quando desconectado
	// case data.StateConnecting:
	// 	hc.BackendActivity.Start() // Inicia a animação ao conectar
	// case data.StateConnected:
	// 	hc.BackendActivity.Start() // Mantém a animação enquanto conectado
	// }
}
