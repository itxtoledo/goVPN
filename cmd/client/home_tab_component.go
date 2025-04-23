// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/home_tab_component.go
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// HomeTabComponent representa o conteúdo da aba Home
type HomeTabComponent struct {
	UI           *UIManager
	Container    *fyne.Container
	NetworkTree  *NetworkTreeComponent
	EmptyContent *fyne.Container
}

// NewHomeTabComponent cria uma nova instância do componente da aba Home
func NewHomeTabComponent(ui *UIManager, networkTree *NetworkTreeComponent) *HomeTabComponent {
	comp := &HomeTabComponent{
		UI:          ui,
		NetworkTree: networkTree,
		Container:   container.NewMax(),
	}

	// Define tamanho máximo para o container
	comp.Container.Resize(fyne.NewSize(280, 500)) // Um pouco menos que o tamanho da janela

	comp.createEmptyContent()
	comp.updateContent()

	return comp
}

// createEmptyContent cria o conteúdo a ser exibido quando não há redes disponíveis
func (h *HomeTabComponent) createEmptyContent() {
	// Criação dos botões para quando não há redes
	createNetButton := widget.NewButton("Criar uma Rede", func() {
		if h.UI.RoomWindow == nil {
			h.UI.RoomWindow = NewRoomWindow(h.UI)
		}
		h.UI.RoomWindow.Show()
	})
	createNetButton.Importance = widget.HighImportance

	connectNetButton := widget.NewButton("Conectar a uma Rede", func() {
		if h.UI.ConnectWindow == nil {
			h.UI.ConnectWindow = NewConnectWindow(h.UI)
		}
		h.UI.ConnectWindow.Show()
	})

	// Limita o tamanho máximo dos botões
	createNetButton.Resize(fyne.NewSize(260, 40))
	connectNetButton.Resize(fyne.NewSize(260, 40))

	// Criando um texto informativo com status online/offline dinâmico
	infoText := "This area will list your networks and peers. You are now " +
		func() string {
			if h.UI.VPN.IsConnected {
				return "online"
			}
			return "offline"
		}() +
		", but this computer is not yet a member in a goVPN network."

	// Texto informativo com alinhamento centralizado
	statusText := widget.NewLabelWithStyle(
		infoText,
		fyne.TextAlignCenter,
		fyne.TextStyle{Italic: true},
	)

	// Limita o tamanho dos textos para evitar quebra de linha não intencional
	statusText.Wrapping = fyne.TextWrapWord
	statusText.Resize(fyne.NewSize(260, 80))

	// Centralizando todos os textos horizontalmente
	title := widget.NewLabelWithStyle("Sem redes disponíveis", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	title.Resize(fyne.NewSize(260, 30))

	description := widget.NewLabelWithStyle("Você não está conectado a nenhuma rede. Escolha uma opção:",
		fyne.TextAlignCenter, fyne.TextStyle{})
	description.Resize(fyne.NewSize(260, 40))
	description.Wrapping = fyne.TextWrapWord

	// Container de botões mais compacto
	buttonContainer := container.NewVBox(
		createNetButton,
		connectNetButton,
	)

	// Conteúdo principal com tamanho controlado
	h.EmptyContent = container.NewVBox(
		title,
		widget.NewSeparator(),
		description,
		buttonContainer,
		widget.NewSeparator(),
		statusText,
	)

	// Define um tamanho fixo para o container vazio
	h.EmptyContent.Resize(fyne.NewSize(280, 400))
}

// updateContent atualiza o conteúdo da aba Home com base no status da conexão
func (h *HomeTabComponent) updateContent() {
	h.Container.Objects = nil // Remove all objects

	if len(h.UI.NetworkUsers) == 0 && !h.UI.VPN.IsConnected {
		// Usando um layout mais simples para evitar expansão indesejada da janela
		h.Container.Add(container.NewCenter(h.EmptyContent))
	} else {
		// Exibe a árvore de rede quando há redes ou está conectado
		h.Container.Add(h.NetworkTree.GetNetworkTree())
	}

	h.Container.Refresh()
}

// GetContainer retorna o container da aba Home
func (h *HomeTabComponent) GetContainer() *fyne.Container {
	return h.Container
}
