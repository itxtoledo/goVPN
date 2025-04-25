// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/home_tab_component.go
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// HomeTabComponent represents the content of the Home tab
type HomeTabComponent struct {
	UI          *UIManager
	Container   *fyne.Container
	NetworkTree *NetworkTreeComponent
}

// NewHomeTabComponent creates a new instance of the Home tab component
func NewHomeTabComponent(ui *UIManager, networkTree *NetworkTreeComponent) *HomeTabComponent {
	comp := &HomeTabComponent{
		UI:          ui,
		NetworkTree: networkTree,
		Container:   container.NewMax(),
	}

	// Define maximum size for the container
	comp.Container.Resize(fyne.NewSize(280, 500)) // A bit smaller than the window size

	// Add the network tree component (it will handle empty state internally)
	comp.Container.Add(networkTree.GetContainer())

	return comp
}

// GetContainer returns the Home tab container
func (h *HomeTabComponent) GetContainer() *fyne.Container {
	return h.Container
}
