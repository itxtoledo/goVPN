package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/pion/webrtc/v3"
)

// SettingsWindow gerencia a interface de configurações
type SettingsWindow struct {
	UI           *UIManager
	Window       fyne.Window
	Container    *fyne.Container
	ServerEntry  *widget.Entry
	EncryptCheck *widget.Check
	StunEntry    *widget.Entry
	SaveButton   *widget.Button
}

// NewSettingsWindow cria uma nova janela de configurações
func NewSettingsWindow(ui *UIManager) *SettingsWindow {
	settingsWindow := &SettingsWindow{
		UI:     ui,
		Window: ui.createWindow("Settings - goVPN", 500, 350, false),
	}

	// Adiciona o manipulador para quando a janela for fechada
	settingsWindow.Window.SetOnClosed(func() {
		settingsWindow.Window = nil
	})

	return settingsWindow
}

// Show exibe a janela de configurações
func (sw *SettingsWindow) Show() {
	// Se a janela já foi fechada, recria
	if sw.Window == nil {
		sw.Window = sw.UI.createWindow("Settings - goVPN", 500, 350, false)
		// Re-adiciona o manipulador para quando a janela for fechada
		sw.Window.SetOnClosed(func() {
			sw.Window = nil
		})
	}

	// Inicializa ou atualiza os componentes antes de exibir
	content := sw.CreateContent()

	// Define o conteúdo da janela
	sw.Window.SetContent(content)

	// Exibe a janela centralizada
	sw.Window.CenterOnScreen()
	sw.Window.Show()
}

// CreateContent cria o conteúdo da janela de configurações
func (sw *SettingsWindow) CreateContent() fyne.CanvasObject {
	// Campos de entrada para configurações
	sw.ServerEntry = widget.NewEntry()
	sw.ServerEntry.SetPlaceHolder("Endereço do servidor de sinalização")

	// Obter o valor atual do servidor de sinalização
	signalServer := "ws://localhost:8080/ws" // Valor padrão
	if sw.UI != nil && sw.UI.VPN != nil && sw.UI.VPN.NetworkManager != nil {
		if sw.UI.VPN.NetworkManager.SignalServer != "" {
			signalServer = sw.UI.VPN.NetworkManager.SignalServer
		}
	}
	sw.ServerEntry.SetText(signalServer)

	sw.StunEntry = widget.NewEntry()
	sw.StunEntry.SetPlaceHolder("Servidor STUN")

	// Valor padrão para o servidor STUN
	stunServer := "stun:stun.l.google.com:19302"

	// Tenta obter o valor atual, com tratamento de erros
	if sw.UI != nil && sw.UI.VPN != nil && sw.UI.VPN.NetworkManager != nil {
		if len(sw.UI.VPN.NetworkManager.ICEServers) > 0 &&
			len(sw.UI.VPN.NetworkManager.ICEServers[0].URLs) > 0 {
			stunServer = sw.UI.VPN.NetworkManager.ICEServers[0].URLs[0]
		}
	}
	sw.StunEntry.SetText(stunServer)

	sw.EncryptCheck = widget.NewCheck("Criptografar tráfego", nil)
	sw.EncryptCheck.SetChecked(true)

	// Formulário de configurações
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Servidor de Sinalização", Widget: sw.ServerEntry},
			{Text: "Servidor STUN", Widget: sw.StunEntry},
			{Text: "Segurança", Widget: sw.EncryptCheck},
		},
		SubmitText: "Salvar",
		OnSubmit: func() {
			sw.saveSettings()
		},
	}

	// Container principal
	content := container.NewVBox(
		widget.NewLabel("Configurações do Cliente"),
		form,
	)

	return content
}

// saveSettings salva as configurações no banco de dados
func (sw *SettingsWindow) saveSettings() {
	// Atualiza as configurações de rede
	sw.UI.VPN.NetworkManager.SignalServer = sw.ServerEntry.Text

	// Atualiza o servidor STUN
	if sw.StunEntry.Text != "" {
		sw.UI.VPN.NetworkManager.ICEServers = []webrtc.ICEServer{
			{
				URLs: []string{sw.StunEntry.Text},
			},
		}
	}

	// Salva as configurações no banco de dados
	_, err := sw.UI.VPN.DB.Exec(`
		INSERT OR REPLACE INTO settings (key, value) VALUES ('signal_server', ?);
	`, sw.ServerEntry.Text)

	if err != nil {
		log.Printf("Erro ao salvar configurações: %v", err)
		sw.UI.ShowMessage("Erro", "Não foi possível salvar as configurações")
		return
	}

	_, err = sw.UI.VPN.DB.Exec(`
		INSERT OR REPLACE INTO settings (key, value) VALUES ('stun_server', ?);
	`, sw.StunEntry.Text)

	if err != nil {
		log.Printf("Erro ao salvar configurações: %v", err)
		sw.UI.ShowMessage("Erro", "Não foi possível salvar as configurações")
		return
	}

	_, err = sw.UI.VPN.DB.Exec(`
		INSERT OR REPLACE INTO settings (key, value) VALUES ('encrypt_traffic', ?);
	`, sw.EncryptCheck.Checked)

	if err != nil {
		log.Printf("Erro ao salvar configurações: %v", err)
		sw.UI.ShowMessage("Erro", "Não foi possível salvar as configurações")
		return
	}

	sw.UI.ShowMessage("Sucesso", "Configurações salvas com sucesso")
}
