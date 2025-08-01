package clientwebrtc_impl

import (
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

// WebRTCManager handles the WebRTC connection and data channel
type WebRTCManager struct {
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel

	// Callbacks
	onConnectionStateChange func(webrtc.PeerConnectionState)
	onICEConnectionStateChange func(webrtc.ICEConnectionState)
	onDataChannelMessage func([]byte)
	onDataChannelOpen func()
}

// NewWebRTCManager creates a new WebRTCManager
func NewWebRTCManager() (*WebRTCManager, error) {
	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	w := &WebRTCManager{
		peerConnection: peerConnection,
	}

	// Set up event handlers for the peer connection
	w.peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if w.onConnectionStateChange != nil {
			w.onConnectionStateChange(s)
		}
	})

	w.peerConnection.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		if w.onICEConnectionStateChange != nil {
			w.onICEConnectionStateChange(s)
		}
	})

	return w, nil
}

// SetOnConnectionStateChange sets the callback for peer connection state changes
func (w *WebRTCManager) SetOnConnectionStateChange(callback func(s webrtc.PeerConnectionState)) {
	w.onConnectionStateChange = callback
}

// SetOnICEConnectionStateChange sets the callback for ICE connection state changes
func (w *WebRTCManager) SetOnICEConnectionStateChange(callback func(s webrtc.ICEConnectionState)) {
	w.onICEConnectionStateChange = callback
}

// SetOnDataChannelMessage sets the callback for data channel messages
func (w *WebRTCManager) SetOnDataChannelMessage(callback func([]byte)) {
	w.onDataChannelMessage = callback
}

// SetOnDataChannelOpen sets the callback for data channel open event
func (w *WebRTCManager) SetOnDataChannelOpen(callback func()) {
	w.onDataChannelOpen = callback
}

// CreateOffer creates an SDP offer to start the connection
func (w *WebRTCManager) CreateOffer(iceRestart bool) (*webrtc.SessionDescription, error) {
	offerOptions := &webrtc.OfferOptions{
		ICERestart: iceRestart,
	}
	offer, err := w.peerConnection.CreateOffer(offerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create offer: %w", err)
	}

	if err = w.peerConnection.SetLocalDescription(offer); err != nil {
		return nil, fmt.Errorf("failed to set local description: %w", err)
	}

	return &offer, nil
}

// HandleOfferAndCreateAnswer handles an incoming SDP offer and creates an SDP answer
func (w *WebRTCManager) HandleOfferAndCreateAnswer(offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	if err := w.peerConnection.SetRemoteDescription(offer); err != nil {
		return nil, fmt.Errorf("failed to set remote description: %w", err)
	}

	answer, err := w.peerConnection.CreateAnswer(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create answer: %w", err)
	}

	if err = w.peerConnection.SetLocalDescription(answer); err != nil {
		return nil, fmt.Errorf("failed to set local description: %w", err)
	}

	return &answer, nil
}

// HandleAnswer handles an incoming SDP answer
func (w *WebRTCManager) HandleAnswer(answer webrtc.SessionDescription) error {
	if err := w.peerConnection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}
	return nil
}

// SetOnICECandidate sets a callback to be notified of new ICE candidates
func (w *WebRTCManager) SetOnICECandidate(callback func(c *webrtc.ICECandidate)) {
	w.peerConnection.OnICECandidate(callback)
}

// AddICECandidate adds a new ICE candidate to the peer connection
func (w *WebRTCManager) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	if err := w.peerConnection.AddICECandidate(candidate); err != nil {
		return fmt.Errorf("failed to add ICE candidate: %w", err)
	}
	return nil
}

// Close closes the WebRTC connection
func (w *WebRTCManager) Close() error {
	if w.dataChannel != nil {
		if err := w.dataChannel.Close(); err != nil {
			return err
		}
	}
	if w.peerConnection != nil {
		if err := w.peerConnection.Close(); err != nil {
			return err
		}
	}
	return nil
}

// CreateDataChannel creates a new data channel and sets up the event handlers
func (w *WebRTCManager) CreateDataChannel() error {
	// Create a new data channel
	dataChannel, err := w.peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		return fmt.Errorf("failed to create data channel: %w", err)
	}

	w.dataChannel = dataChannel

	// Set up the event handlers
	w.dataChannel.OnOpen(func() {
		log.Println("Data channel opened")
		if w.onDataChannelOpen != nil {
			w.onDataChannelOpen()
		}
	})

	w.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Message from data channel: %s\n", string(msg.Data))
		if w.onDataChannelMessage != nil {
			w.onDataChannelMessage(msg.Data)
		}
	})

	return nil
}

// SendMessage sends a message over the data channel
func (w *WebRTCManager) SendMessage(message string) error {
	if w.dataChannel == nil || w.dataChannel.ReadyState() != webrtc.DataChannelStateOpen {
		return fmt.Errorf("data channel is not open")
	}

	return w.dataChannel.SendText(message)
}


