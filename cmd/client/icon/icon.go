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

// Credits: https://www.svgrepo.com/collection/wolf-kit-solid-glyph-icons/3
// Logo: https://www.svgrepo.com/svg/10943/umbrella
// License: https://www.svgrepo.com/page/licensing/#CC%20Attribution
//
// All icons were modified, the color and some properties of SVG files were altered.
// https://www.svgrepo.com/svg/91531/vpn?edit=true
var (
	LinkOn  = PrepareResource("link.svg")
	LinkOff = PrepareResource("link_off.svg")
	Power   = PrepareResource("power.svg")
	VPN     = PrepareResource("vpn.svg")
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
