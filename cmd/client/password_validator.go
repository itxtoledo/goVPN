package main

import (
	"errors"
	"regexp"

	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/libs/models"
)

// ConfigurePasswordEntry configura um campo de entrada de senha com validação de 4 dígitos
// e aplicação automática de regras de formato de senha (somente 4 dígitos numéricos)
func ConfigurePasswordEntry(passwordEntry *widget.Entry) {
	// Define o validador para verificar a senha usando o validador do pacote models
	passwordEntry.Validator = func(s string) error {
		if !models.ValidatePassword(s) {
			return errors.New("password must be exactly 4 digits")
		}
		return nil
	}

	// Configura o manipulador OnChanged para filtrar entrada e limitar a 4 dígitos
	passwordEntry.OnChanged = func(s string) {
		// Filtra caracteres não-numéricos
		re := regexp.MustCompile("[^0-9]")
		filteredText := re.ReplaceAllString(s, "")

		// Trunca para máximo de 4 caracteres
		if len(filteredText) > 4 {
			filteredText = filteredText[:4]
		}

		// Só atualiza se o texto realmente mudou para evitar loop infinito
		if filteredText != s {
			passwordEntry.SetText(filteredText)
		}
	}
}

// ValidatePassword verifica se uma senha atende ao padrão definido no pacote models
func ValidatePassword(password string) bool {
	return models.ValidatePassword(password)
}
