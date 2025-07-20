package main

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"

	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/dialogs"
	"github.com/itxtoledo/govpn/cmd/client/icon"
)

// NetworkListComponent representa o componente da árvore de rede
type NetworkListComponent struct {
	UI               *UIManager
	Container        *fyne.Container
	NetworkAccordion *CustomAccordion
	contentContainer *fyne.Container // New field to hold dynamic content
	updateMutex      sync.Mutex
}

// NewNetworkListComponent cria uma nova instância do componente de árvore de rede
func NewNetworkListComponent(ui *UIManager) *NetworkListComponent {
	ntc := &NetworkListComponent{
		UI: ui,
	}
	ntc.init()
	return ntc
}

// init inicializa o componente
func (ntc *NetworkListComponent) init() {
	// Criar um accordion personalizado vazio
	ntc.NetworkAccordion = NewCustomAccordion()
	// Initialize the dynamic content container
	ntc.contentContainer = container.NewStack()

	// Criar o container principal
	ntc.Container = container.NewBorder(
		nil, // widget.NewLabelWithStyle("Available Networks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: false}),
		nil,
		nil,
		nil,
		ntc.contentContainer, // Use the new contentContainer here
	)
}

// UpdateNetworkList atualiza a lista de redes
func (ntc *NetworkListComponent) UpdateNetworkList() {
	// Use mutex to prevent concurrent modifications
	ntc.updateMutex.Lock()
	defer ntc.updateMutex.Unlock()

	// Clear the content container before adding new content
	ntc.contentContainer.RemoveAll()

	// Sort the networks by name
	networks := ntc.UI.RealtimeData.GetNetworks()
	if networks != nil {
		sort.Slice(networks, func(i, j int) bool {
			return networks[i].Name < networks[j].Name
		})
	}

	log.Printf("Updating network list. Total: %d", len(networks))

	if len(networks) > 0 {
		// Clear the accordion before adding new items
		ntc.NetworkAccordion.RemoveAll()

		// Get the current network ID from the network manager (if connected)
		currentNetworkID := ""
		if ntc.UI.VPN.NetworkManager != nil {
			currentNetworkID = ntc.UI.VPN.NetworkManager.NetworkID
		}

		// Get computername from config for display
		computername, _ := ntc.UI.RealtimeData.ComputerName.Get()
		if computername == "" {
			computername = "You"
		}

		// Add each network as an accordion item
		for _, network := range networks {
			log.Printf("Processing network: %s (ID=%s)", network.Name, network.ID)
			// Check if this network is the one we're currently connected to
			isConnected := network.ID == currentNetworkID

			// Create connected computers list
			computersContainer := container.NewVBox()

			// Add current computer to the list (always show if we're in this network)
			var currentStatusResource = icon.ConnectionOff
			if isConnected {
				currentStatusResource = icon.ConnectionOn
			}

			var displayMyComputerName string
			if len(computername) > 5 {
				displayMyComputerName = computername[:5] + "..."
			} else {
				displayMyComputerName = computername
			}

			myComputerIP, _ := ntc.UI.RealtimeData.ComputerIP.Get()
			currentComputerItem := container.NewHBox(
				widget.NewIcon(currentStatusResource),
				widget.NewLabelWithStyle(displayMyComputerName+" (you)", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
				layout.NewSpacer(),
				widget.NewLabelWithStyle(myComputerIP, fyne.TextAlignTrailing, fyne.TextStyle{Monospace: true}),
			)
			computersContainer.Add(currentComputerItem)

			// Add other computers in the network
			// Add other computers in the network
			if ntc.UI.VPN.NetworkManager != nil && len(ntc.UI.VPN.NetworkManager.Computers) > 0 {
				// Get computers in the network from the NetworkManager
				for _, computer := range ntc.UI.VPN.NetworkManager.Computers {
					// Skip our own computer
					if computer.OwnerID == ntc.UI.VPN.PublicKeyStr {
						continue
					}

					// Create activity indicator based on online status
					var activity = icon.ConnectionOff
					if computer.IsOnline {
						activity = icon.ConnectionOn
					}

					// Create computer item with icon, activity indicator and name
					// Create computer item with icon, activity indicator and name
					var displayComputerName string
					if len(computer.Name) > 5 {
						displayComputerName = computer.Name[:5] + "..."
					} else {
						displayComputerName = computer.Name
					}
					computerItem := container.NewHBox(
						widget.NewIcon(activity),
						widget.NewLabelWithStyle(displayComputerName, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
						layout.NewSpacer(),
						widget.NewLabelWithStyle(computer.PeerIP, fyne.TextAlignTrailing, fyne.TextStyle{Monospace: true}),
					)
					computersContainer.Add(computerItem)
				}
			}

			// Create computers section
			computersBox := computersContainer

			// Create actions section
			actionsBox := container.NewVBox()

			content := container.NewHBox(
				container.NewVBox(
					computersBox,
					widget.NewSeparator(),
					actionsBox,
				),
			)

			// Create custom title with activity indicator or radio button icon
			var statusIndicator = icon.ConnectionOff
			if isConnected {
				statusIndicator = icon.ConnectionOn
			}

			// Calculate connected computers count
			connectedComputers := 0
			if isConnected {
				connectedComputers = 1 // Count yourself if connected
			}

			// Count other computers that are online
			if ntc.UI.VPN.NetworkManager != nil && len(ntc.UI.VPN.NetworkManager.Computers) > 0 {
				for _, computer := range ntc.UI.VPN.NetworkManager.Computers {
					if computer.OwnerID != ntc.UI.VPN.PublicKeyStr && computer.IsOnline {
						connectedComputers++
					}
				}
			}

			titleLabel := widget.NewLabelWithStyle(network.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			computerCountLabel := widget.NewLabelWithStyle(fmt.Sprintf("(%d/10)", connectedComputers), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

			customTitle := container.NewHBox(
				widget.NewIcon(statusIndicator),
				titleLabel,
			)

			// Create custom accordion item with context menu support and computer count
			accordionItem := NewCustomAccordionItemWithEndContentAndCallbacks(customTitle, content, computerCountLabel, nil, func(pe *fyne.PointEvent) {
				leaveItem := fyne.NewMenuItem("Leave Network", func() {
					// Delegate deletion to NetworkManager
					if ntc.UI.VPN.NetworkManager != nil {
						go func() {
							err := ntc.UI.VPN.NetworkManager.LeaveNetworkById(network.ID)
							if err != nil {
								log.Printf("Error deleting network: %v", err)
								fyne.CurrentApp().SendNotification(&fyne.Notification{
									Title:   "Error",
									Content: "Failed to leave network: " + err.Error(),
								})
							} else {
								log.Println("Successfully left network:", network.Name)
								fyne.CurrentApp().SendNotification(&fyne.Notification{
									Title:   "Success",
									Content: "Successfully left network: " + network.Name,
								})

								// Show success dialog on the main thread
								dialog.ShowInformation("Success", "Successfully left network: "+network.Name, ntc.UI.MainWindow)
							}
						}()
					}
				})

				connectItemLabel := "Connect"
				if isConnected {
					connectItemLabel = "Disconnect"
				}
				connectItem := fyne.NewMenuItem(connectItemLabel, func() {
					ntc.UI.SelectedNetwork = network

					if isConnected {
						// If already connected, disconnect
						log.Println("Disconnecting from network:", network.Name)
						go func() {
							err := ntc.UI.VPN.NetworkManager.DisconnectNetwork(network.ID)
							if err != nil {
								log.Printf("Error disconnecting from network: %v", err)
								dialog.ShowError(fmt.Errorf("failed to disconnect from network: %v", err), ntc.UI.MainWindow)
							} else {
								log.Println("Successfully disconnected from network.")
								dialog.ShowInformation("Success", "Successfully disconnected from network.", ntc.UI.MainWindow)
							}
						}()
					} else {
						// Show connection dialog
						if ntc.UI.ConnectDialog == nil {
							ntc.UI.ConnectDialog = dialogs.NewConnectDialog(ntc.UI, ntc.UI.VPN.ComputerName)
						}
						ntc.UI.ConnectDialog.Show()
					}
				})

				menu := fyne.NewMenu(network.Name, connectItem, leaveItem)
				popUp := widget.NewPopUpMenu(menu, ntc.UI.MainWindow.Canvas())
				popUp.ShowAtPosition(pe.AbsolutePosition)
			})

			// Auto-open if connected
			if isConnected {
				accordionItem.Open()
			}

			// Add item to accordion
			ntc.NetworkAccordion.AddItem(accordionItem)
		}

		if len(ntc.NetworkAccordion.Items) > 0 {
			// Add the accordion container to the content container
			ntc.contentContainer.Add(ntc.NetworkAccordion.GetContainer())
		} else {
			log.Printf("No networks available to display after filtering/processing") // Add informative message when no networks are available
			noNetworksLabel := widget.NewLabelWithStyle(
				"No networks available.\nCreate or join a network to get started.",
				fyne.TextAlignCenter,
				fyne.TextStyle{Italic: true},
			)
			ntc.contentContainer.Add(container.NewCenter(noNetworksLabel)) // Add centered label
		}
	} else {
		log.Printf("No networks available to display")
		// Add informative message when no networks are available
		noNetworksLabel := widget.NewLabelWithStyle(
			"No networks available.\nCreate or join a network to get started.",
			fyne.TextAlignCenter,
			fyne.TextStyle{Italic: true},
		)
		ntc.contentContainer.Add(container.NewCenter(noNetworksLabel)) // Add centered label
	}

	// Refresh the content container and main container
	ntc.contentContainer.Refresh()
	ntc.Container.Refresh()
}

// GetContainer retorna o container principal
func (ntc *NetworkListComponent) GetContainer() *fyne.Container {
	return ntc.Container
}
