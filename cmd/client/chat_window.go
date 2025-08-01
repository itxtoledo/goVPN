package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/ui"
	
)

// ChatWindow represents the chat window for a network
type ChatWindow struct {
	ui.BaseWindow
	network       *data.Network
	onSendMessage   func(peerPublicKey string, message string) error
	onMessageReceived func(peerPublicKey string, message string)
	clientPublicKey string
	messageEntry  *widget.Entry
	messageList   *widget.List
	messages      []string
}

var globalChatWindow *ChatWindow

// NewChatWindow creates a new instance of the chat window
func NewChatWindow(app fyne.App, network *data.Network, onSendMessage func(peerPublicKey string, message string) error, onMessageReceived func(peerPublicKey string, message string), clientPublicKey string) *ChatWindow {
	cw := &ChatWindow{
		network:       network,
		onSendMessage: onSendMessage,
		onMessageReceived: onMessageReceived,
		clientPublicKey: clientPublicKey,
		messages:      []string{},
	}
			cw.BaseWindow = *ui.NewBaseWindow(app, network.NetworkName+" Chat", 400, 500)
	cw.BaseWindow.Window.SetFixedSize(true)
	cw.BaseWindow.Window.SetOnClosed(func() {
		globalChatWindow = nil
	})
	cw.setupUI()
	return cw
}

// setupUI initializes the UI components of the chat window
func (cw *ChatWindow) setupUI() {
	cw.messageList = widget.NewList(
		func() int {
			return len(cw.messages)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(cw.messages[i])
		},
	)

	cw.messageEntry = widget.NewEntry()
	cw.messageEntry.SetPlaceHolder("Type a message...")

	sendButton := widget.NewButton("Send", cw.sendMessage)

	inputContainer := container.NewBorder(
		nil,
		nil,
		nil,
		sendButton,
		cw.messageEntry,
	)

	content := container.NewBorder(
		nil,
		inputContainer,
		nil,
		nil,
		cw.messageList,
	)

	cw.BaseWindow.Window.SetContent(content)

	// Set up the callback for incoming messages
	if cw.onMessageReceived != nil {
		// This callback will be triggered by the UIManager's handleIncomingWebRTCMessage
		// We need a way to pass the message to this specific chat window instance.
		// For now, the UIManager directly calls addMessage on globalChatWindow.
		// This section is more for conceptual understanding of where a direct callback would go.
	}
}

// sendMessage sends a message via WebRTC
func (cw *ChatWindow) sendMessage() {
	message := cw.messageEntry.Text
	if message == "" {
		return
	}

	// TODO: Implement sending message via WebRTC
	// For now, just add to the list
	cw.addMessage("You: " + message)
	cw.messageEntry.SetText("")

	// Send the message using WebRTC
	// TODO: Determine the target peer(s) for the message
	// For now, let's assume we send to all connected peers in the network
	// This will require iterating through the network.Computers and calling onSendMessage for each.
	// For simplicity in this refactor, let's assume a single target for now or a broadcast mechanism
	// that will be handled by the NetworkManager or a higher level.
	// For the purpose of getting the chat working, we'll just call onSendMessage with a placeholder peer.
	// The actual peer selection logic will need to be implemented based on UI/UX.

	// Placeholder: Assuming we send to the first computer in the network (excluding self)
	var targetPeerPublicKey string
	for _, computer := range cw.network.Computers {
		// Need to get our own public key to exclude ourselves
		// This information is not directly available in ChatWindow, it should come from VPNClient/ConfigManager
		// For now, let's assume a dummy public key for testing.
		// In a real scenario, you'd pass the current client's public key to ChatWindow.
		if computer.PublicKey != cw.clientPublicKey { // Use the actual client public key
			targetPeerPublicKey = computer.PublicKey
			break
		}
	}

	if targetPeerPublicKey == "" {
		log.Println("No target peer found to send message.")
		cw.addMessage("Error: No peer to send message to.")
		return
	}

	err := cw.onSendMessage(targetPeerPublicKey, message)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		cw.addMessage("Error sending: " + err.Error())
	}
}

// addMessage adds a message to the chat display
func (cw *ChatWindow) addMessage(message string) {
	fyne.Do(func() {
		cw.messages = append(cw.messages, message)
		cw.messageList.Refresh()
		// Scroll to bottom
		cw.messageList.ScrollToBottom()
	})
}

// Show shows the chat window
func (cw *ChatWindow) Show() {
	cw.BaseWindow.Show()
}
