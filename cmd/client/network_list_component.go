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
	"github.com/itxtoledo/govpn/cmd/client/ui"
)

// NetworkListComponent representa o componente da árvore de rede
type NetworkListComponent struct {
	UI               *UIManager
	Container        *fyne.Container
	NetworkAccordion *ui.CustomAccordion
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
	ntc.NetworkAccordion = ui.NewCustomAccordion()
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

	fyne.Do(func() {
		// Clear the content container before adding new content
		ntc.contentContainer.RemoveAll()

		// Sort the networks by name
		networks := ntc.UI.RealtimeData.GetNetworks()
		if networks != nil {
			sort.Slice(networks, func(i, j int) bool {
				return networks[i].NetworkName < networks[j].NetworkName
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
				log.Printf("Processing network: %s (ID=%s)", network.NetworkName, network.NetworkID)
				// Check if this network is the one we're currently connected to
				// Use a copy of the network for the closure to avoid unexpected behavior
				// due to loop variable reuse.
				localNetwork := network
				isConnected := localNetwork.NetworkID == currentNetworkID

				// Create connected computers list
				computersContainer := container.NewVBox()

				// Use apenas os computadores que vêm do servidor

				// Get our public key for comparison
				myPublicKey := ""
				if ntc.UI.VPN != nil {
					myPublicKey = ntc.UI.VPN.PublicKeyStr
				}

				// Add all computers from the network response
				if len(localNetwork.Computers) > 0 {
					for _, computer := range localNetwork.Computers {

						// Create activity indicator based on online status
						var activity = icon.ConnectionOff

						// Se este computador for o nosso e estivermos conectados a esta rede,
						// mostrar como conectado independentemente do status online
						if isConnected && myPublicKey != "" && computer.PublicKey == myPublicKey {
							activity = icon.ConnectionOn
						} else if computer.IsOnline {
							activity = icon.ConnectionOn
						}

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
							widget.NewLabelWithStyle(computer.ComputerIP, fyne.TextAlignTrailing, fyne.TextStyle{Monospace: true}),
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
						actionsBox,
					),
				)

				// Create custom title without activity indicator
				// Status indicator code removed

				// Calculate connected computers count - use only computers from server response
				connectedComputers := 0

				// Count computers that are online
				if localNetwork.Computers != nil {
					for _, computer := range localNetwork.Computers {
						if computer.IsOnline {
							connectedComputers++
						}
					}
				}
				titleLabel := widget.NewLabelWithStyle(localNetwork.NetworkName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
				computerCountLabel := widget.NewLabelWithStyle(fmt.Sprintf("(%d/10)", connectedComputers), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

				customTitle := container.NewHBox(
					titleLabel,
				)

				// Create custom accordion item with context menu support and computer count
				accordionItem := ui.NewCustomAccordionItemWithEndContentAndCallbacks(customTitle, content, computerCountLabel, nil, func(pe *fyne.PointEvent) {
					leaveItem := fyne.NewMenuItem("Leave Network", func() {
						// Delegate deletion to NetworkManager
						if ntc.UI.VPN.NetworkManager != nil {
							go func() {
								err := ntc.UI.VPN.NetworkManager.LeaveNetworkById(localNetwork.NetworkID)
								if err != nil {
									log.Printf("Error deleting network: %v", err)
									fyne.CurrentApp().SendNotification(&fyne.Notification{
										Title:   "Error",
										Content: "Failed to leave network: " + err.Error(),
									})
								} else {
									log.Println("Successfully left network:", localNetwork.NetworkName)
									fyne.CurrentApp().SendNotification(&fyne.Notification{
										Title:   "Success",
										Content: "Successfully left network: " + localNetwork.NetworkName,
									})

									// Show success dialog on the main thread
									dialog.ShowInformation("Success", "Successfully left network: "+localNetwork.NetworkName, ntc.UI.MainWindow)
								}
							}()
						}
					})

					connectItemLabel := "Connect"
					if isConnected {
						connectItemLabel = "Disconnect"
					}
					connectItem := fyne.NewMenuItem(connectItemLabel, func() {
						ntc.UI.SelectedNetwork = &localNetwork

						if isConnected {
							// If already connected, disconnect
							log.Println("Disconnecting from network:", localNetwork.NetworkName)
							go func() {
								err := ntc.UI.VPN.NetworkManager.DisconnectNetwork(localNetwork.NetworkID)
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

					menu := fyne.NewMenu(localNetwork.NetworkName, connectItem, leaveItem)
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
	})
}

// GetContainer retorna o container principal
func (ntc *NetworkListComponent) GetContainer() *fyne.Container {
	return ntc.Container
}
