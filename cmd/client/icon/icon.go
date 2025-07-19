/*
Package icon provides compiled icons for the application.
*/
package icon

import (
	"embed"
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed *
var assetsFS embed.FS

var (
	LinkOn        = PrepareResource("link.svg")
	LinkOff       = PrepareResource("link_off.svg")
	Power         = PrepareResource("power.svg")
	VPN           = PrepareResource("vpn.svg")
	AppIcon       = PrepareResource("app.png")
	ConnectionOn  = PrepareResource("connection_on.svg")
	ConnectionOff = PrepareResource("connection_off.svg")
)

func init() {
}

func PrepareResource(path string) fyne.Resource {
	b, err := assetsFS.ReadFile("assets/" + path)
	if err != nil {
		panic(err)
	}

	return &fyne.StaticResource{StaticName: path, StaticContent: b}
}
