package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// ShowMessage displays a simple message dialog to the user
func ShowMessage(title, message string, window fyne.Window) {
	dialog.ShowInformation(title, message, window)
}

// ShowError displays an error message dialog to the user
func ShowError(err error, window fyne.Window) {
	dialog.ShowError(err, window)
}
