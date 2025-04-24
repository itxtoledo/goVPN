package main

import (
	"runtime/debug"

	"fyne.io/systray"
)

const (
	AppTitleName = "goVPN"
)

// var translations embed.FS

func initialize() {
	debug.SetGCPercent(10)
	// root.PromptRootAccess()
}

func onstart() {
	systray.SetTooltip(AppTitleName)
	// dock.HideIconInDock()
}

func main() {
	vpn := NewVPNClient()
	initialize()
	vpn.UI.App.Lifecycle().SetOnStarted(onstart)
	defer vpn.DB.Close()

	vpn.Run()
}
