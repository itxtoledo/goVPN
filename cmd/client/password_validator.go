package main

import (
	"errors"
	"regexp"

	"fyne.io/fyne/v2/widget"
)

// ConfigurePasswordEntry configura um campo de entrada de senha com validação de 4 dígitos
// e aplicação automática de regras de formato de senha (somente 4 dígitos numéricos)
func ConfigurePasswordEntry(passwordEntry *widget.Entry) {
	// Define o validador para verificar que a senha tem exatamente 4 dígitos numéricos
	passwordEntry.Validator = func(s string) error {
		if len(s) != 4 {
			return errors.New("Password must be exactly 4 digits")
		}
		matched, _ := regexp.MatchString("^[0-9]{4}$", s)
		if !matched {
			return errors.New("Password must contain only numbers")
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

// ValidatePassword verifica se uma senha atende ao requisito de 4 dígitos numéricos
func ValidatePassword(password string) bool {
	if len(password) != 4 {
		return false
	}

	matched, _ := regexp.MatchString("^[0-9]{4}$", password)
	return matched
}
