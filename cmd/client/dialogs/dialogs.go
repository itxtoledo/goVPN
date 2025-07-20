package dialogs

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ShowMessage displays a simple message dialog to the computer
func ShowMessage(title, message string, window fyne.Window) {
	dialog.ShowInformation(title, message, window)
}

// ShowError displays an error message dialog to the computer
func ShowError(err error, window fyne.Window) {
	dialog.ShowError(err, window)
}

// isDigit checks if a string contains only digits
func isDigit(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ValidatePIN checks if a PIN matches the default PIN pattern
func ValidatePIN(pin string) bool {
	return len(pin) == 4 && isDigit(pin)
}

// ConfigurePINEntry configures a PIN entry widget with 4-digit validation
func ConfigurePINEntry(entry *widget.Entry) {
	entry.SetPlaceHolder("4-digit PIN")
	entry.Validator = func(s string) error {
		if !ValidatePIN(s) {
			return errors.New("PIN must be exactly 4 digits")
		}
		return nil
	}
}
