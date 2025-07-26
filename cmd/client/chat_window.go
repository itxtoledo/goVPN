package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/itxtoledo/govpn/cmd/client/data"
	"github.com/itxtoledo/govpn/cmd/client/ui"
	"github.com/itxtoledo/govpn/cmd/client/webrtc"
)

// ChatWindow represents the chat window for a network
type ChatWindow struct {
	ui.BaseWindow
	network       *data.Network
	webrtcManager *clientwebrtc_impl.WebRTCManager
	messageEntry  *widget.Entry
	messageList   *widget.List
	messages      []string
}

var globalChatWindow *ChatWindow

// NewChatWindow creates a new instance of the chat window
func NewChatWindow(app fyne.App, network *data.Network, webrtcManager *clientwebrtc_impl.WebRTCManager) *ChatWindow {
	cw := &ChatWindow{
		network:       network,
		webrtcManager: webrtcManager,
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

	// Listen for incoming WebRTC messages
	cw.webrtcManager.SetOnMessageReceived(func(message string) {
		cw.addMessage("Received: " + message)
	})
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
	err := cw.webrtcManager.SendMessage(message)
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
