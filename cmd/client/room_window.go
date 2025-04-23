package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// RoomWindow gerencia a interface de salas da VPN
type RoomWindow struct {
	UI                *UIManager
	Window            fyne.Window
	RoomList          *widget.List
	Rooms             []models.Room
	CreateBtn         *widget.Button
	JoinBtn           *widget.Button
	RefreshBtn        *widget.Button
	Container         *fyne.Container
	selectedRoomIndex int // Armazena o índice da sala selecionada
}

// NewRoomWindow cria uma nova janela de salas
func NewRoomWindow(ui *UIManager) *RoomWindow {
	roomWindow := &RoomWindow{
		UI:                ui,
		Rooms:             []models.Room{},
		Window:            ui.createWindow("Salas - goVPN", 600, 400, false),
		selectedRoomIndex: -1, // Nenhuma sala selecionada inicialmente
	}

	// Configura a atualização da lista de salas
	ui.VPN.NetworkManager.OnRoomListUpdate = roomWindow.updateRoomList

	return roomWindow
}

// Show exibe a janela de salas
func (rw *RoomWindow) Show() {
	// Se a janela já foi fechada, recria
	if rw.Window == nil {
		rw.Window = rw.UI.createWindow("Salas - goVPN", 600, 400, false)
	}

	// Certifica-se de que a janela está com o conteúdo atualizado
	if rw.Window.Content() == nil {
		rw.Window.SetContent(rw.CreateContent())
	}

	// Atualiza a lista de salas
	rw.refreshRoomList()

	rw.Window.Show()
	rw.Window.CenterOnScreen()
}

// CreateContent cria o conteúdo da janela de salas
func (rw *RoomWindow) CreateContent() fyne.CanvasObject {
	// Lista de salas
	rw.RoomList = widget.NewList(
		func() int { return len(rw.Rooms) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Nome da sala"),
				widget.NewLabel("Usuários"),
				widget.NewLabel("Status"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			room := rw.Rooms[id]
			labels := item.(*fyne.Container).Objects
			statusText := "Disponível"
			if room.IsCreator {
				statusText = "Criada por você"
			}

			labels[0].(*widget.Label).SetText(room.Name)
			labels[1].(*widget.Label).SetText(fmt.Sprintf("%d", room.ClientCount))
			labels[2].(*widget.Label).SetText(statusText)
		},
	)

	// Adiciona seleção à lista de salas
	rw.RoomList.OnSelected = func(id widget.ListItemID) {
		// Armazena o índice selecionado
		rw.selectedRoomIndex = id
		// Habilita o botão de entrar quando uma sala é selecionada
		rw.JoinBtn.Enable()
	}

	// Adiciona um handler para retirar a seleção
	rw.RoomList.OnUnselected = func(id widget.ListItemID) {
		rw.selectedRoomIndex = -1
		rw.JoinBtn.Disable()
	}

	// Botões de ação
	rw.CreateBtn = widget.NewButton("Criar Sala", rw.showCreateRoomDialog)
	rw.JoinBtn = widget.NewButton("Entrar", rw.showJoinRoomDialog)
	rw.JoinBtn.Disable() // Inicialmente desabilitado até que uma sala seja selecionada

	rw.RefreshBtn = widget.NewButton("Atualizar Lista", rw.refreshRoomList)

	buttonBar := container.NewHBox(
		rw.CreateBtn,
		rw.JoinBtn,
		rw.RefreshBtn,
	)

	// Container principal
	rw.Container = container.NewBorder(
		widget.NewLabel("Salas disponíveis:"),
		buttonBar,
		nil, nil,
		container.NewScroll(rw.RoomList),
	)

	return rw.Container
}

// updateRoomList atualiza a lista de salas na interface
func (rw *RoomWindow) updateRoomList(rooms []models.Room) {
	rw.Rooms = rooms
	rw.RoomList.Refresh()
	rw.selectedRoomIndex = -1 // Reinicia o índice selecionado
	rw.JoinBtn.Disable()      // Reinicia o estado do botão de entrar
}

// refreshRoomList solicita uma atualização da lista de salas ao servidor
func (rw *RoomWindow) refreshRoomList() {
	// Verifique se o gerenciador de rede está pronto
	if rw.UI.VPN.NetworkManager == nil {
		log.Println("Erro: NetworkManager não foi inicializado")
		return
	}

	// Tenta atualizar a lista de salas em uma goroutine separada
	go func() {
		// Tenta conectar caso não esteja conectado
		if !rw.UI.VPN.NetworkManager.IsConnected {
			err := rw.UI.VPN.NetworkManager.Connect()
			if err != nil {
				// Registra o erro no log
				log.Printf("Erro de conexão: %v", err)

				 // Use fyne.Do para executar o código na thread principal da UI
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Erro de Conexão",
					Content: "Não foi possível conectar ao servidor de sinalização",
				})
				return
			}
		}

		// Realiza a operação de rede
		err := rw.UI.VPN.NetworkManager.GetRoomList()
		if err != nil {
			log.Printf("Erro ao obter lista de salas: %v", err)

			 // Use fyne.Do para executar o código na thread principal da UI
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Erro",
				Content: "Não foi possível atualizar a lista de salas",
			})
		}
	}()
}

// showCreateRoomDialog exibe o diálogo para criar uma nova sala
func (rw *RoomWindow) showCreateRoomDialog() {
	// Verificar se está conectado ao backend
	if !rw.UI.VPN.NetworkManager.IsConnected {
		// Tenta conectar ao servidor em uma goroutine separada
		go func() {
			err := rw.UI.VPN.NetworkManager.Connect()
			// Voltar para a thread principal para atualizar a UI
			if err != nil {
				// Use SendNotification para exibir uma notificação de erro
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Erro de Conexão",
					Content: "Não foi possível conectar ao servidor de sinalização",
				})
			} else {
				// Se conectou com sucesso, usamos a janela principal para mostrar o diálogo
				// De volta à thread principal
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Conexão Estabelecida",
					Content: "Conexão estabelecida com sucesso",
				})
				// Precisamos usar a thread principal para UI
				// Como não podemos usar Driver().Run, vamos tentar criar o diálogo
				// na próxima vez que o usuário clicar no botão
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "Pronto",
					Content: "Clique em Criar Sala novamente para continuar",
				})
			}
		}()
	} else {
		// Já está conectado, continuar normalmente
		rw.showCreateRoomDialogAfterConnect()
	}
}

// showCreateRoomDialogAfterConnect exibe o diálogo depois de confirmar conexão ao servidor
func (rw *RoomWindow) showCreateRoomDialogAfterConnect() {
	// Verificar se o usuário já tem uma chave pública
	if rw.UI.VPN.PublicKeyPEM == "" {
		// Se não existe chave pública, tenta gerá-la
		err := rw.UI.VPN.loadOrGenerateKeys()
		if err != nil {
			rw.UI.ShowMessage("Erro", "Não foi possível criar ou carregar suas chaves de identificação: "+err.Error())
			return
		}
	}

	// Implementar diálogo de criação de sala
	roomNameEntry := widget.NewEntry()
	roomPasswordEntry := widget.NewPasswordEntry()

	// Limitar a senha para apenas números com máximo de 4 dígitos
	roomPasswordEntry.Validator = func(s string) error {
		if len(s) > 4 {
			return fmt.Errorf("a senha deve ter no máximo 4 caracteres")
		}
		for _, r := range s {
			if r < '0' || r > '9' {
				return fmt.Errorf("a senha deve conter apenas números")
			}
		}
		return nil
	}

	// Limitar entrada para 4 caracteres em tempo real
	roomPasswordEntry.OnChanged = func(s string) {
		if len(s) > 4 {
			roomPasswordEntry.SetText(s[:4])
		}
	}

	content := container.NewVBox(
		widget.NewLabel("Nome da Sala:"),
		roomNameEntry,
		widget.NewLabel("Senha (4 dígitos):"),
		roomPasswordEntry,
		widget.NewLabel("Nota: Cada usuário só pode criar uma sala.\nSe você já criou uma sala, não poderá criar outra."),
	)

	// Criamos o popup modal
	popup := widget.NewModalPopUp(
		container.NewVBox(
			content,
			container.NewHBox(
				widget.NewButton("Cancelar", func() {
					if rw.Window.Canvas().Overlays().Top() != nil {
						rw.Window.Canvas().Overlays().Top().Hide()
					}
				}),
				widget.NewButton("Criar", func() {
					// Validar campos
					if roomNameEntry.Text == "" || roomPasswordEntry.Text == "" {
						rw.UI.ShowMessage("Erro", "Preencha todos os campos")
						return
					}

					// Validar tamanho da senha
					if len(roomPasswordEntry.Text) < 4 {
						rw.UI.ShowMessage("Erro", "A senha deve ter exatamente 4 dígitos")
						return
					}

					// Criar sala em uma goroutine separada
					go func() {
						err := rw.UI.VPN.NetworkManager.CreateRoom(roomNameEntry.Text, roomPasswordEntry.Text)
						if err != nil {
							// Registra o erro e mostra uma notificação
							log.Printf("Erro ao criar sala: %v", err)
							fyne.CurrentApp().SendNotification(&fyne.Notification{
								Title:   "Erro",
								Content: "Não foi possível criar a sala",
							})
						} else {
							// Sucesso - apenas atualiza a lista na thread principal
							// Como não podemos usar Driver().Run diretamente,
							// usamos SendNotification e faremos o cliente clicar em Atualizar
							fyne.CurrentApp().SendNotification(&fyne.Notification{
								Title:   "Sucesso",
								Content: "Sala criada com sucesso! Clique em Atualizar Lista.",
							})
						}
					}()
				}),
			),
		),
		rw.Window.Canvas(),
	)

	popup.Show()
}

// showJoinRoomDialog exibe o diálogo para entrar em uma sala
func (rw *RoomWindow) showJoinRoomDialog() {
	// Verificar se uma sala foi selecionada
	if rw.selectedRoomIndex < 0 || rw.selectedRoomIndex >= len(rw.Rooms) {
		rw.UI.ShowMessage("Erro", "Selecione uma sala primeiro")
		return
	}

	selectedRoom := rw.Rooms[rw.selectedRoomIndex]
	roomPasswordEntry := widget.NewPasswordEntry()

	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Entrar na sala: %s", selectedRoom.Name)),
		widget.NewLabel("Senha:"),
		roomPasswordEntry,
	)

	// Criamos o popup modal
	popup := widget.NewModalPopUp(
		container.NewVBox(
			content,
			container.NewHBox(
				widget.NewButton("Cancelar", func() {
					if rw.Window.Canvas().Overlays().Top() != nil {
						rw.Window.Canvas().Overlays().Top().Hide()
					}
				}),
				widget.NewButton("Entrar", func() {
					// Validar campos
					if roomPasswordEntry.Text == "" {
						rw.UI.ShowMessage("Erro", "Informe a senha da sala")
						return
					}

					// Entrar na sala
					err := rw.UI.VPN.NetworkManager.JoinRoom(selectedRoom.ID, roomPasswordEntry.Text)
					if err != nil {
						rw.UI.ShowMessage("Erro", "Não foi possível entrar na sala: "+err.Error())
					} else {
						// Atualiza o status de conexão
						rw.UI.VPN.CurrentRoom = selectedRoom.ID
						rw.UI.VPN.IsConnected = true
						rw.UI.updatePowerButtonState()
					}
					if rw.Window.Canvas().Overlays().Top() != nil {
						rw.Window.Canvas().Overlays().Top().Hide()
					}
				}),
			),
		),
		rw.Window.Canvas(),
	)

	popup.Show()
}
