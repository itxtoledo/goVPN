// filepath: /Users/gustavotoledodesouza/Projects/fun/goVPN/cmd/client/network_tree_component.go
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NetworkTreeComponent representa a árvore de visualização de rede
type NetworkTreeComponent struct {
	UI          *UIManager
	NetworkList *widget.Tree
}

// NewNetworkTreeComponent cria uma nova instância do componente de árvore de rede
func NewNetworkTreeComponent(ui *UIManager) *NetworkTreeComponent {
	comp := &NetworkTreeComponent{
		UI: ui,
	}

	comp.setupNetworkTree()
	return comp
}

// setupNetworkTree configura a árvore de rede que mostra usuários
func (n *NetworkTreeComponent) setupNetworkTree() {
	n.NetworkList = widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			if id == "" {
				return []widget.TreeNodeID{"network"}
			} else if id == "network" {
				userIDs := make([]widget.TreeNodeID, 0, len(n.UI.NetworkUsers))
				for userID := range n.UI.NetworkUsers {
					userIDs = append(userIDs, "user_"+userID)
				}
				return userIDs
			}
			return []widget.TreeNodeID{}
		},
		func(id widget.TreeNodeID) bool {
			return id == "" || id == "network"
		},
		func(branch bool) fyne.CanvasObject {
			if branch {
				// For the network node, place status icon on the left, label in the center, and expand icon on the right
				statusIcon := widget.NewIcon(theme.RadioButtonIcon())
				label := widget.NewLabel("Network")
				expandIcon := widget.NewIcon(theme.MenuDropDownIcon())
				// Use container.NewHBox to ensure proper ordering
				return container.NewHBox(
					statusIcon,
					label,
					layout.NewSpacer(),
					expandIcon,
				)
			}
			// For users, keep the simple format
			return container.NewHBox(
				widget.NewIcon(theme.ComputerIcon()),
				widget.NewLabel("User"),
			)
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			if branch {
				// Update elements for the network node
				container := o.(*fyne.Container)
				statusIcon := container.Objects[0].(*widget.Icon) // Status icon at index 0
				label := container.Objects[1].(*widget.Label)     // Label at index 1
				expandIcon := container.Objects[3].(*widget.Icon) // Expand icon at index 3

				if id == "network" {
					label.SetText("Network")

					// Update status icon based on connection
					if n.UI.VPN.IsConnected {
						statusIcon.SetResource(theme.NewColoredResource(theme.RadioButtonCheckedIcon(), theme.ColorNameSuccess))
					} else {
						statusIcon.SetResource(theme.NewColoredResource(theme.RadioButtonIcon(), theme.ColorNameError))
					}

					// Update expand/collapse icon based on state
					if n.NetworkList.IsBranchOpen(id) {
						expandIcon.SetResource(theme.MenuDropUpIcon())
					} else {
						expandIcon.SetResource(theme.MenuDropDownIcon())
					}
				}
			} else if id[:5] == "user_" {
				container := o.(*fyne.Container)
				label := container.Objects[1].(*widget.Label)
				icon := container.Objects[0].(*widget.Icon)

				userID := id[5:]
				isOnline := n.UI.NetworkUsers[userID]

				ipText := "Offline user"
				if isOnline {
					if n.UI.VPN.NetworkManager.VirtualNetwork != nil {
						// Using the network library's implementation
						// Get peer's virtual IP if exists
						ipText = "Online user"
						icon.SetResource(theme.RadioButtonCheckedIcon())
					} else {
						ipText = "Online user"
						icon.SetResource(theme.RadioButtonCheckedIcon())
					}
				} else {
					icon.SetResource(theme.RadioButtonIcon())
				}

				label.SetText(ipText)
			}
		},
	)

	// Expand the network node by default
	n.NetworkList.OpenBranch("network")

	// Add listeners to update expand/collapse icons
	n.NetworkList.OnBranchOpened = func(id widget.TreeNodeID) {
		if id == "network" {
			n.NetworkList.Refresh()
		}
	}

	n.NetworkList.OnBranchClosed = func(id widget.TreeNodeID) {
		if id == "network" {
			n.NetworkList.Refresh()
		}
	}
}

// GetNetworkTree retorna o widget de árvore de rede
func (n *NetworkTreeComponent) GetNetworkTree() *fyne.Container {
	// Garantir que a árvore de rede não expanda além do tamanho da janela
	container := container.NewMax(n.NetworkList)
	container.Resize(fyne.NewSize(280, 500))
	return container
}

// RefreshTree atualiza a visualização da árvore
func (n *NetworkTreeComponent) RefreshTree() {
	n.NetworkList.Refresh()
}
