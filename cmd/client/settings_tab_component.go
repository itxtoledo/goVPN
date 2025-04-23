// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/settings_tab_component.go
package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/pion/webrtc/v3"
)

// SettingsTabComponent representa o componente da aba Configurações
type SettingsTabComponent struct {
	UI           *UIManager
	Container    *fyne.Container
	ServerEntry  *widget.Entry
	StunEntry    *widget.Entry
	EncryptCheck *widget.Check
	SaveButton   *widget.Button
}

// NewSettingsTabComponent cria uma nova instância do componente da aba Configurações
func NewSettingsTabComponent(ui *UIManager) *SettingsTabComponent {
	comp := &SettingsTabComponent{
		UI: ui,
	}

	comp.createContent()
	return comp
}

// createContent cria o conteúdo da aba Configurações
func (s *SettingsTabComponent) createContent() {
	// Campos de configuração
	s.ServerEntry = widget.NewEntry()
	s.ServerEntry.SetPlaceHolder("Endereço do servidor de sinalização")

	// Obter o valor atual do servidor de sinalização
	signalServer := "wss://govpn-signal.example.com/ws" // Valor padrão
	if s.UI.VPN.NetworkManager.SignalServer != "" {
		signalServer = s.UI.VPN.NetworkManager.SignalServer
	}
	s.ServerEntry.SetText(signalServer)

	s.StunEntry = widget.NewEntry()
	s.StunEntry.SetPlaceHolder("Servidor STUN")

	// Valor padrão para o servidor STUN
	stunServer := "stun:stun.l.google.com:19302"
	if len(s.UI.VPN.NetworkManager.ICEServers) > 0 &&
		len(s.UI.VPN.NetworkManager.ICEServers[0].URLs) > 0 {
		stunServer = s.UI.VPN.NetworkManager.ICEServers[0].URLs[0]
	}
	s.StunEntry.SetText(stunServer)

	s.EncryptCheck = widget.NewCheck("Criptografar tráfego", nil)
	s.EncryptCheck.SetChecked(true)

	s.SaveButton = widget.NewButton("Salvar Configurações", func() {
		s.saveSettings()
	})

	// Container principal
	s.Container = container.NewVBox(
		widget.NewLabelWithStyle("Configurações", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Servidor de Sinalização", s.ServerEntry),
			widget.NewFormItem("Servidor STUN", s.StunEntry),
			widget.NewFormItem("Segurança", s.EncryptCheck),
		),
		s.SaveButton,
	)
}

// saveSettings salva as configurações do usuário
func (s *SettingsTabComponent) saveSettings() {
	// Atualiza as configurações de rede
	s.UI.VPN.NetworkManager.SignalServer = s.ServerEntry.Text

	// Atualiza o servidor STUN
	if s.StunEntry.Text != "" {
		s.UI.VPN.NetworkManager.ICEServers = []webrtc.ICEServer{
			{
				URLs: []string{s.StunEntry.Text},
			},
		}
	}

	// Salva as configurações no banco de dados
	_, err := s.UI.VPN.DB.Exec(`
		INSERT OR REPLACE INTO settings (key, value) VALUES ('signal_server', ?);
	`, s.ServerEntry.Text)

	if err != nil {
		log.Printf("Erro ao salvar configurações: %v", err)
		s.UI.ShowMessage("Erro", "Não foi possível salvar as configurações")
		return
	}

	_, err = s.UI.VPN.DB.Exec(`
		INSERT OR REPLACE INTO settings (key, value) VALUES ('stun_server', ?);
	`, s.StunEntry.Text)

	if err != nil {
		log.Printf("Erro ao salvar configurações: %v", err)
		s.UI.ShowMessage("Erro", "Não foi possível salvar as configurações")
		return
	}

	_, err = s.UI.VPN.DB.Exec(`
		INSERT OR REPLACE INTO settings (key, value) VALUES ('encrypt_traffic', ?);
	`, s.EncryptCheck.Checked)

	if err != nil {
		log.Printf("Erro ao salvar configurações: %v", err)
		s.UI.ShowMessage("Erro", "Não foi possível salvar as configurações")
		return
	}

	s.UI.ShowMessage("Sucesso", "Configurações salvas com sucesso")
}

// GetContainer retorna o container da aba Configurações
func (s *SettingsTabComponent) GetContainer() *fyne.Container {
	return s.Container
}
