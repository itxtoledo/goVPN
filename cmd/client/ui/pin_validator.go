package ui

import (
	"errors"
	"regexp"

	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/utils"
)

// ConfigurePINEntry configura um campo de entrada de PIN com validação de 4 dígitos
// e aplicação automática de regras de formato de PIN (somente 4 dígitos numéricos)
func ConfigurePINEntry(pinEntry *widget.Entry) {
	// Define o validador para verificar o PIN usando o validador do pacote models
	pinEntry.Validator = func(s string) error {
		if !utils.ValidatePIN(s) {
			return errors.New("PIN must be exactly 4 digits")
		}
		return nil
	}

	// Configura o manipulador OnChanged para filtrar entrada e limitar a 4 dígitos
	pinEntry.OnChanged = func(s string) {
		// Filtra caracteres não-numéricos
		re := regexp.MustCompile("[^0-9]")
		filteredText := re.ReplaceAllString(s, "")

		// Trunca para máximo de 4 caracteres
		if len(filteredText) > 4 {
			filteredText = filteredText[:4]
		}

		// Só atualiza se o texto realmente mudou para evitar loop infinito
		if filteredText != s {
			pinEntry.SetText(filteredText)
		}
	}
}
