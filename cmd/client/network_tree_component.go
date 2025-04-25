// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/network_tree_component.go
package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// NetworkTreeComponent representa a lista scrollable de salas
type NetworkTreeComponent struct {
	ui               *UIManager
	localRooms       []models.Room                 // Lista de salas locais
	container        *fyne.Container               // Container principal
	scrollContainer  *container.Scroll             // Container com scroll
	roomsList        *fyne.Container               // Container de lista vertical
	roomComponents   []*RoomItemComponent          // Lista de componentes de sala
	roomComponentMap map[string]*RoomItemComponent // Mapa de componentes por ID
	emptyContent     *fyne.Container               // Conteúdo quando não há salas disponíveis
}

// NewNetworkTreeComponent cria um novo componente de lista de salas
func NewNetworkTreeComponent(ui *UIManager) *NetworkTreeComponent {
	ntc := &NetworkTreeComponent{
		ui:               ui,
		localRooms:       []models.Room{},
		roomComponents:   []*RoomItemComponent{},
		roomComponentMap: make(map[string]*RoomItemComponent),
	}

	// Cria um container vertical para a lista de salas
	ntc.roomsList = container.NewVBox()

	// Cria container com barra de rolagem
	ntc.scrollContainer = container.NewScroll(ntc.roomsList)
	ntc.container = container.NewMax(ntc.scrollContainer)

	// Cria o conteúdo para quando não há salas
	ntc.createEmptyContent()

	// Registra callback para atualizar quando a lista de salas mudar
	if ui != nil && ui.VPN != nil && ui.VPN.NetworkManager != nil {
		ui.VPN.NetworkManager.OnRoomListUpdate = func(rooms []models.Room) {
			// Atualiza as salas locais
			ntc.updateLocalRooms(rooms)
		}
	}

	// Carrega salas locais iniciais
	ntc.loadLocalRooms()

	return ntc
}

// createEmptyContent cria o conteúdo a ser exibido quando não há salas disponíveis
func (ntc *NetworkTreeComponent) createEmptyContent() {
	// Criando botões para quando não há redes
	createNetButton := widget.NewButton("Criar uma Sala", func() {
		if ntc.ui.RoomDialog == nil {
			ntc.ui.RoomDialog = NewRoomDialog(ntc.ui)
		}
		ntc.ui.RoomDialog.Show()
	})
	createNetButton.Importance = widget.HighImportance

	connectNetButton := widget.NewButton("Conectar a uma Sala", func() {
		if ntc.ui.ConnectDialog == nil {
			ntc.ui.ConnectDialog = NewConnectDialog(ntc.ui)
		}
		ntc.ui.ConnectDialog.Show()
	})

	// Limita o tamanho máximo dos botões
	createNetButton.Resize(fyne.NewSize(260, 40))
	connectNetButton.Resize(fyne.NewSize(260, 40))

	// Criando um texto informativo com status online/offline dinâmico
	infoText := "Esta área vai mostrar suas salas salvas. Você está " +
		func() string {
			if ntc.ui.VPN.IsConnected {
				return "conectado"
			}
			return "desconectado"
		}() +
		", mas ainda não tem salas salvas localmente."

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
	title := widget.NewLabelWithStyle("Nenhuma sala disponível", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	title.Resize(fyne.NewSize(260, 30))

	description := widget.NewLabelWithStyle("Você não está conectado a nenhuma sala. Escolha uma opção:",
		fyne.TextAlignCenter, fyne.TextStyle{})
	description.Resize(fyne.NewSize(260, 40))
	description.Wrapping = fyne.TextWrapWord

	// Container de botões mais compacto
	buttonContainer := container.NewVBox(
		createNetButton,
		connectNetButton,
	)

	// Conteúdo principal com tamanho controlado
	ntc.emptyContent = container.NewVBox(
		title,
		widget.NewSeparator(),
		description,
		buttonContainer,
		widget.NewSeparator(),
		statusText,
	)

	// Define um tamanho fixo para o container vazio
	ntc.emptyContent.Resize(fyne.NewSize(280, 400))
}

// GetContainer retorna o container do componente
func (ntc *NetworkTreeComponent) GetContainer() *fyne.Container {
	return ntc.container
}

// loadLocalRooms carrega salas do banco de dados local
func (ntc *NetworkTreeComponent) loadLocalRooms() {
	// Verifica se o VPN client e NetworkManager estão disponíveis
	if ntc.ui == nil || ntc.ui.VPN == nil || ntc.ui.VPN.NetworkManager == nil {
		log.Printf("Não foi possível carregar salas locais: UI ou NetworkManager é nil")
		return
	}

	localRooms, err := ntc.ui.VPN.NetworkManager.loadLocalRooms()
	if err != nil {
		log.Printf("Erro ao carregar salas locais: %v", err)
		return
	}

	// Atualiza lista de salas locais
	ntc.localRooms = localRooms

	ntc.updateRoomsList()

}

// updateLocalRooms atualiza a lista com novas salas
func (ntc *NetworkTreeComponent) updateLocalRooms(serverRooms []models.Room) {
	// Se serverRooms for nil, recarrega apenas as salas locais
	if serverRooms == nil {
		ntc.loadLocalRooms()
		return
	}

	// Carrega salas locais novamente para garantir sincronização
	localRooms, err := ntc.ui.VPN.NetworkManager.loadLocalRooms()
	if err != nil {
		log.Printf("Erro ao carregar salas locais: %v", err)
		return
	}

	// Atualiza a lista de salas locais
	ntc.localRooms = localRooms

	// Atualiza a lista de salas
	ntc.updateRoomsList()
}

// updateRoomsList atualiza a lista visual de salas
func (ntc *NetworkTreeComponent) updateRoomsList() {
	// Limpa o container principal
	ntc.container.Objects = nil

	// Limpa a lista de salas
	ntc.roomsList.Objects = nil
	ntc.roomComponents = nil
	ntc.roomComponentMap = make(map[string]*RoomItemComponent)

	// Verifica se há salas para mostrar
	if len(ntc.localRooms) > 0 {
		// Adiciona cada sala como um componente individual
		for _, room := range ntc.localRooms {
			// Cria um novo componente de sala
			roomComponent := NewRoomItemComponent(ntc.ui, room)

			// Adiciona à lista de componentes
			ntc.roomComponents = append(ntc.roomComponents, roomComponent)
			ntc.roomComponentMap[room.ID] = roomComponent

			// Adiciona o container do componente à lista visual
			ntc.roomsList.Add(roomComponent.GetContainer())
		}

		// Exibe a lista de salas
		ntc.container.Add(ntc.scrollContainer)
	} else {
		// Se não há salas, exibe o conteúdo vazio
		ntc.container.Add(container.NewCenter(ntc.emptyContent))
	}

	// Refresh para atualizar a interface
	ntc.container.Refresh()
	ntc.roomsList.Refresh()
}

// updateRoomItems atualiza o estado de todos os componentes de sala
func (ntc *NetworkTreeComponent) updateRoomItems() {
	for _, roomComponent := range ntc.roomComponents {
		roomComponent.Update()
	}
}

// getConnectedRoomComponent retorna o componente da sala conectada, se houver
func (ntc *NetworkTreeComponent) getConnectedRoomComponent() *RoomItemComponent {
	if ntc.ui.VPN.IsConnected && ntc.ui.VPN.CurrentRoom != "" {
		if component, exists := ntc.roomComponentMap[ntc.ui.VPN.CurrentRoom]; exists {
			return component
		}
	}
	return nil
}

// Refresh atualiza explicitamente a lista de redes
func (ntc *NetworkTreeComponent) Refresh() {
	ntc.loadLocalRooms()
}
