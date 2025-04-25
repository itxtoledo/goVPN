// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/room_item_component.go
package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// RoomItemComponent representa um item individual de sala com acordeão
type RoomItemComponent struct {
	ui           *UIManager
	Room         models.Room
	Container    *fyne.Container
	Accordion    *widget.Accordion
	isConnected  bool
	usersContent *fyne.Container
}

// NewRoomItemComponent cria um novo componente de item de sala
func NewRoomItemComponent(ui *UIManager, room models.Room) *RoomItemComponent {
	ric := &RoomItemComponent{
		ui:          ui,
		Room:        room,
		isConnected: ui.VPN.IsConnected && ui.VPN.CurrentRoom == room.ID,
	}

	// Cria um container para os usuários
	ric.usersContent = container.NewVBox()

	// Atualiza os usuários no container
	ric.updateUsersContent()

	// Cria um acordeão para a sala
	ric.Accordion = widget.NewAccordion(
		widget.NewAccordionItem(ric.getRoomTitle(), ric.usersContent),
	)

	// Cria o container principal
	ric.Container = container.NewMax(ric.Accordion)

	// Configura comportamento ao clicar no acordeão
	ric.setupAccordionBehavior()

	return ric
}

// getRoomTitle retorna o título da sala (com indicador de conexão se estiver conectado)
func (ric *RoomItemComponent) getRoomTitle() string {
	if ric.isConnected {
		return ric.Room.Name + " (Conectado)"
	}
	return ric.Room.Name
}

// setupAccordionBehavior configura o comportamento do acordeão ao ser clicado
func (ric *RoomItemComponent) setupAccordionBehavior() {
	// Verificação de segurança para evitar nil pointer
	if ric == nil || ric.ui == nil || ric.Accordion == nil || len(ric.Accordion.Items) == 0 || ric.Container == nil {
		return
	}

	// Se não estiver conectado, adiciona um botão de conexão ao conteúdo
	if !ric.isConnected {
		// Armazena o ID e senha da sala
		roomID := ric.Room.ID
		roomPassword := ric.Room.Password

		// Cria um botão de conexão
		connectButton := widget.NewButton("Conectar a esta sala", func() {
			// Verificação de segurança
			if ric == nil || ric.ui == nil || ric.ui.VPN == nil {
				return
			}

			// Se já estiver conectado a outra sala, pergunta se deseja mudar
			if ric.ui.VPN.IsConnected {
				var parentWindow fyne.Window
				if ric.ui.MainWindow != nil {
					parentWindow = ric.ui.MainWindow
				} else {
					windows := fyne.CurrentApp().Driver().AllWindows()
					if len(windows) > 0 {
						parentWindow = windows[0]
					} else {
						ric.connectToRoom(roomID, roomPassword)
						return
					}
				}

				confirmDialog := dialog.NewConfirm(
					"Trocar sala",
					"Você já está conectado a uma sala. Deseja desconectar e conectar a esta sala?",
					func(confirmed bool) {
						if confirmed {
							ric.connectToRoom(roomID, roomPassword)
						}
					},
					parentWindow,
				)
				confirmDialog.Show()
			} else {
				// Conectar diretamente
				ric.connectToRoom(roomID, roomPassword)
			}
		})

		// Adiciona o botão ao conteúdo do usuário
		ric.usersContent.Add(connectButton)
		ric.usersContent.Refresh()
	}
}

// updateUsersContent atualiza o conteúdo de usuários no acordeão
func (ric *RoomItemComponent) updateUsersContent() {
	// Verificação de segurança para evitar nil pointer
	if ric.usersContent == nil {
		ric.usersContent = container.NewVBox()
	}

	// Limpa o container de usuários
	ric.usersContent.Objects = nil

	if ric.isConnected {
		// Adiciona o usuário atual com estilo em negrito e indicador de conexão
		userLabel := widget.NewLabelWithStyle(
			"Você (Conectado)",
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		)
		ric.usersContent.Add(userLabel)

		// Adiciona um ícone de conexão
		connectionIcon := widget.NewIcon(theme.ConfirmIcon())
		ric.usersContent.Add(connectionIcon)

		// Adiciona outros usuários se houver
		for username := range ric.ui.NetworkUsers {
			if username != "Você" {
				peerLabel := widget.NewLabel(username)
				ric.usersContent.Add(peerLabel)
			}
		}

		// Adiciona um botão para desconectar
		disconnectButton := widget.NewButton("Disconnect", func() {
			if ric.ui.VPN.NetworkManager != nil {
				err := ric.ui.VPN.NetworkManager.LeaveRoom()
				if err != nil {
					log.Printf("Erro ao desconectar: %v", err)
					ric.ui.ShowMessage("Erro", "Não foi possível desconectar da sala.")
				}
			}
		})
		ric.usersContent.Add(disconnectButton)
	} else {
		// Mesmo quando não conectado, mostra o usuário atual como membro da sala
		userLabel := widget.NewLabelWithStyle(
			"Você",
			fyne.TextAlignLeading,
			fyne.TextStyle{Italic: true},
		)
		ric.usersContent.Add(userLabel)

		// Adiciona uma mensagem informativa sobre a conexão
		infoLabel := widget.NewLabelWithStyle(
			"Conecte-se para ver outros membros",
			fyne.TextAlignCenter,
			fyne.TextStyle{Italic: true},
		)
		ric.usersContent.Add(infoLabel)

		// Adiciona um botão para se conectar à sala
		connectButton := widget.NewButton("Connect", func() {
			// Armazena o ID e senha da sala localmente para uso
			roomID := ric.Room.ID
			roomPassword := ric.Room.Password

			// Tenta conectar à sala
			err := ric.ui.VPN.NetworkManager.JoinRoom(roomID, roomPassword)
			if err != nil {
				log.Printf("Erro ao conectar: %v", err)

				// Verifica se o erro indica que a sala não existe mais
				if err.Error() == "Room does not exist" || err.Error() == "erro ao entrar na sala: \"Room does not exist\"" {
					// Remove a sala do banco local
					if ric.ui.VPN.DBManager != nil {
						delErr := ric.ui.VPN.DBManager.DeleteRoom(roomID)
						if delErr != nil {
							log.Printf("Erro ao excluir sala do banco local: %v", delErr)
						}

						// Mostra mensagem sobre a sala removida
						ric.ui.ShowMessage("Sala removida", "Esta sala não existe mais no servidor e foi removida da sua lista.")

						// Atualiza a lista de salas
						if ric.ui.NetworkTreeComponent != nil {
							ric.ui.NetworkTreeComponent.Refresh()
						}
					}
				} else {
					// Outro tipo de erro
					ric.ui.ShowMessage("Erro", "Não foi possível conectar à sala: "+err.Error())
				}
			}
		})
		ric.usersContent.Add(connectButton)

		// Adiciona um separador visual
		ric.usersContent.Add(widget.NewSeparator())

		// Adiciona um botão para sair da sala (excluir da lista local)
		leaveButton := widget.NewButton("Sair da sala", func() {
			roomID := ric.Room.ID

			// Confirmação antes de excluir
			var parentWindow fyne.Window
			if ric.ui.MainWindow != nil {
				parentWindow = ric.ui.MainWindow
			} else {
				windows := fyne.CurrentApp().Driver().AllWindows()
				if len(windows) > 0 {
					parentWindow = windows[0]
				} else {
					// Se não há janela disponível, exclui diretamente
					ric.removeRoomFromLocal(roomID)
					return
				}
			}

			// Diálogo de confirmação
			confirmDialog := dialog.NewConfirm(
				"Sair da sala",
				"Tem certeza que deseja remover esta sala da sua lista? Você poderá entrar novamente se souber o ID e senha.",
				func(confirmed bool) {
					if confirmed {
						ric.removeRoomFromLocal(roomID)
					}
				},
				parentWindow,
			)
			confirmDialog.Show()
		})
		ric.usersContent.Add(leaveButton)
	}

	// Refresh para atualizar a interface
	ric.usersContent.Refresh()

	// Atualiza o título do acordeão somente se ele existir
	if ric.Accordion != nil && len(ric.Accordion.Items) > 0 {
		ric.Accordion.Items[0].Title = ric.getRoomTitle()
		ric.Accordion.Refresh()
	}
}

// connectToRoom tenta conectar a esta sala
func (ric *RoomItemComponent) connectToRoom(roomID, password string) {
	// Desconectar da sala atual se já estiver conectado
	if ric.ui.VPN.IsConnected {
		if err := ric.ui.VPN.NetworkManager.LeaveRoom(); err != nil {
			log.Printf("Erro ao desconectar: %v", err)
			ric.ui.ShowMessage("Erro", "Não foi possível desconectar da sala atual.")
			return
		}
	}

	// Conectar à nova sala
	if err := ric.ui.VPN.NetworkManager.JoinRoom(roomID, password); err != nil {
		log.Printf("Erro ao conectar: %v", err)
		ric.ui.ShowMessage("Erro", "Não foi possível conectar à sala: "+err.Error())
		return
	}

	// A atualização da interface será feita via events do NetworkManager
}

// Update atualiza o estado do componente
func (ric *RoomItemComponent) Update() {
	// Atualiza o estado de conexão
	ric.isConnected = ric.ui.VPN.IsConnected && ric.ui.VPN.CurrentRoom == ric.Room.ID

	// Atualiza o conteúdo de usuários
	ric.updateUsersContent()

	// Refresh do acordeão se existir
	if ric.Accordion != nil {
		ric.Accordion.Refresh()

		// Se estiver conectado, expande o acordeão
		if ric.isConnected && len(ric.Accordion.Items) > 0 {
			ric.Accordion.Open(0)
		}
	} else {
		// Se estamos usando o container customizado, atualize-o
		ric.Container.Refresh()
	}
}

// GetContainer retorna o container do componente
func (ric *RoomItemComponent) GetContainer() *fyne.Container {
	return ric.Container
}

// removeRoomFromLocal exclui a sala do banco de dados local
func (ric *RoomItemComponent) removeRoomFromLocal(roomID string) {
	// Verificação de segurança para evitar nil pointer
	if ric == nil || ric.ui == nil || ric.ui.VPN == nil || ric.ui.VPN.DBManager == nil {
		return
	}

	// Se estiver conectado a esta sala, desconecta primeiro
	if ric.isConnected && ric.ui.VPN.CurrentRoom == roomID {
		if err := ric.ui.VPN.NetworkManager.LeaveRoom(); err != nil {
			log.Printf("Erro ao desconectar antes de remover sala: %v", err)
			// Continuamos com a remoção mesmo se a desconexão falhar
		}
	}

	// Remove a sala do banco de dados local
	err := ric.ui.VPN.DBManager.DeleteRoom(roomID)
	if err != nil {
		log.Printf("Erro ao excluir sala do banco local: %v", err)
		ric.ui.ShowMessage("Erro", "Não foi possível remover a sala da sua lista.")
		return
	}

	// Mostra mensagem de sucesso
	ric.ui.ShowMessage("Sala removida", "A sala foi removida da sua lista com sucesso.")

	// Atualiza a lista de salas
	if ric.ui.NetworkTreeComponent != nil {
		ric.ui.NetworkTreeComponent.Refresh()

		// Verifica se essa era a última sala
		// Obtém a lista atualizada de salas após a remoção
		localRooms, err := ric.ui.VPN.NetworkManager.loadLocalRooms()
		if err == nil && len(localRooms) == 0 {
			// Se não há mais salas, atualiza a interface principal para mostrar os botões de criação/conexão
			if ric.ui.HomeTabComponent != nil {
				// Força a atualização do conteúdo do HomeTabComponent para mostrar os botões
				// ric.ui.HomeTabComponent.updateContent()
			}
		}
	}
}
