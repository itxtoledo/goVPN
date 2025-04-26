// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/home_tab_component.go
package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// HomeTabComponent representa o componente da aba principal
type HomeTabComponent struct {
	UI                *UIManager
	NetworkStatsLabel *widget.Label
	PeerCountLabel    *widget.Label
	LatencyLabel      *widget.Label
	TransferLabel     *widget.Label
}

// NewHomeTabComponent cria uma nova instância do componente da aba principal
func NewHomeTabComponent(ui *UIManager) *HomeTabComponent {
	htc := &HomeTabComponent{
		UI: ui,
	}
	htc.init()
	return htc
}

// init inicializa o componente
func (htc *HomeTabComponent) init() {
	// Inicializar labels
	htc.NetworkStatsLabel = widget.NewLabelWithStyle("Network Statistics", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	htc.PeerCountLabel = widget.NewLabel("Peers: 0")
	htc.LatencyLabel = widget.NewLabel("Latency: 0 ms")
	htc.TransferLabel = widget.NewLabel("Transfer: 0 B sent, 0 B received")

	// Configurar listeners para os dados em tempo real
	htc.UI.RealtimeData.PeerCount.AddListener(binding.NewDataListener(func() {
		count, _ := htc.UI.RealtimeData.PeerCount.Get()
		htc.PeerCountLabel.SetText(fmt.Sprintf("Peers: %d", count))
	}))

	htc.UI.RealtimeData.NetworkLatency.AddListener(binding.NewDataListener(func() {
		latency, _ := htc.UI.RealtimeData.NetworkLatency.Get()
		htc.LatencyLabel.SetText(fmt.Sprintf("Latency: %.2f ms", latency))
	}))

	htc.UI.RealtimeData.TransferredBytes.AddListener(binding.NewDataListener(func() {
		sent, _ := htc.UI.RealtimeData.TransferredBytes.Get()
		received, _ := htc.UI.RealtimeData.ReceivedBytes.Get()
		htc.TransferLabel.SetText(fmt.Sprintf("Transfer: %s sent, %s received",
			formatBytes(int64(sent)), formatBytes(int64(received))))
	}))
}

// formatBytes formata bytes para uma representação legível
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// CreateHomeTabContainer cria o container principal da aba inicial
func (htc *HomeTabComponent) CreateHomeTabContainer() *fyne.Container {
	// Criar o container de estatísticas de rede
	statsContainer := container.NewVBox(
		htc.NetworkStatsLabel,
		widget.NewSeparator(),
		htc.PeerCountLabel,
		htc.LatencyLabel,
		htc.TransferLabel,
	)

	// Criar o container de salas disponíveis
	roomsContainer := htc.UI.NetworkTreeComp.GetContainer()

	// Criar um botão para criar uma nova sala
	createRoomButton := widget.NewButtonWithIcon("Create Room", fyne.Theme.Icon(fyne.CurrentApp().Settings().Theme(), "contentAdd"), func() {
		// Mostrar diálogo de criação de sala
		log.Println("Create room button clicked")
		// Implementar diálogo de criação de sala
	})

	// Criar o container da aba principal
	mainContainer := container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), createRoomButton),
		nil,
		nil,
		container.NewVSplit(
			roomsContainer,
			container.NewPadded(statsContainer),
		),
	)

	return mainContainer
}

// updateUI atualiza a interface do usuário
func (htc *HomeTabComponent) updateUI() {
	// As atualizações já são feitas pelos listeners dos bindings
}
