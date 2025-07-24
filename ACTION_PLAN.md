# goVPN - Full VPN Implementation Plan

This document outlines the necessary steps to evolve goVPN from a signaling/discovery application into a full computer-to-computer VPN client, similar to Hamachi, using WebRTC and a virtual network interface (TUN).

---

## Section 0: Architectural Refinement (Client-Side)

This section focuses on abstracting and centralizing the VPN connection management logic to improve modularity and prepare for WebRTC and TUN integration.

- [ ] **0.1: Define VPNConnectionManager Interface**
    - **Component**: Client (`cmd/client/vpn_connection_manager.go`)
    - **Details**: Create a new package/file to define the core interface for managing VPN connections. This will encapsulate the state and operations related to the VPN. The goal is to have a single, well-defined entry point for all VPN-related actions and state queries.
    - **Action**:
        - Create the file `cmd/client/vpn_connection_manager.go`.
        - Define a `VPNConnectionManager` struct. This struct should contain fields to hold the current state of the VPN, such as:
            - `currentNetworkID string`: The ID of the network the client is currently connected to.
            - `assignedIP net.IP`: The virtual IP address assigned to this client within the VPN network.
            - `peerConnections map[string]*webrtc.PeerConnection`: A map to store active WebRTC peer connections, keyed by the public key of the remote computer. (This will be populated in later tasks).
            - `signalingClient *signaling.Client`: A reference to the signaling client to send/receive messages from the server.
            - `tunInterface *water.Interface`: A reference to the TUN interface (will be populated in Section 1).
            - `mu sync.Mutex`: A mutex to protect concurrent access to the manager's state.
        - Define methods for the `VPNConnectionManager` struct, initially as empty stubs or with basic logging:
            - `NewVPNConnectionManager(signalingClient *signaling.Client) *VPNConnectionManager`: Constructor to initialize the manager.
            - `ConnectToNetwork(networkID string, password string) error`: Initiates a connection to a specified VPN network.
            - `Disconnect() error`: Disconnects from the current VPN network, cleaning up resources.
            - `HandleSignalingMessage(msg models.Message)`: A central handler for all incoming messages from the signaling server. This method will dispatch messages to appropriate internal handlers based on `msg.Type`.
            - `GetConnectionStatus() string`: Returns the current status of the VPN connection (e.g., "Disconnected", "Connecting", "Connected").

- [ ] **0.2: Implement Basic VPNConnectionManager**
    - **Component**: Client (`cmd/client/vpn_connection_manager.go`)
    - **Details**: Implement the basic structure and functionality of the `VPNConnectionManager` to manage the lifecycle of the VPN client. This step focuses on making the manager functional for initial connection and disconnection flows, without full WebRTC or TUN integration yet.
    - **Action**:
        - Implement the `NewVPNConnectionManager()` constructor to properly initialize all fields of the `VPNConnectionManager` struct, including the mutex and the map.
        - Implement `ConnectToNetwork(networkID string, password string) error`:
            - Send a `JoinNetwork` message to the signaling server via `signalingClient`.
            - Update the internal state to "Connecting".
        - Implement `Disconnect() error`:
            - Send a `LeaveNetwork` message to the signaling server.
            - Update the internal state to "Disconnected".
            - Log the action.
        - Implement `HandleSignalingMessage(msg models.Message)`:
            - Use a `switch` statement on `msg.Type` to handle different message types.
            - For `models.NetworkJoined`, update `currentNetworkID` and `assignedIP`. Log the assigned IP.
            - For `models.NetworkLeft`, reset `currentNetworkID` and `assignedIP`.
            - For `models.ComputerJoined` and `models.ComputerLeft`, log the event (full handling will come in Section 2).
            - For `models.SdpOffer`, `models.SdpAnswer`, `models.IceCandidate`, log the event (full handling will come in Section 2).
        - Implement `GetConnectionStatus() string` to return a meaningful status based on the internal state.

- [ ] **0.3: Integrate Signaling Client with VPNConnectionManager**
    - **Component**: Client (`cmd/client/vpn_client.go`, `cmd/client/vpn_connection_manager.go`, `cmd/client/ui_manager.go`)
    - **Details**: The `VPNConnectionManager` should become the primary recipient and dispatcher for all messages received from the signaling server. This centralizes message handling and decouples the UI from direct signaling client interactions.
    - **Action**:
        - In `cmd/client/vpn_client.go` (or `main.go` if `vpn_client.go` is the main entry point):
            - Instantiate the `VPNConnectionManager` early in the application's lifecycle, passing the `signaling.Client` instance to its constructor.
            - Modify the existing message receiving loop (where messages from the WebSocket server are processed) to forward *all* incoming `models.Message` objects directly to `vpnConnectionManager.HandleSignalingMessage(msg)`.
            - Remove any direct message handling logic from `vpn_client.go` that is now handled by the manager.
        - Update `cmd/client/ui_manager.go` (and potentially `home_tab_component.go` or `network_manager.go` if they directly interact with signaling):
            - Modify UI actions (e.g., "Connect" button click, "Disconnect" button click) to call methods on the `VPNConnectionManager` (e.g., `vpnConnectionManager.ConnectToNetwork(...)`, `vpnConnectionManager.Disconnect()`) instead of directly interacting with the signaling client.
            - Update UI elements that display connection status to query `vpnConnectionManager.GetConnectionStatus()`.

- [ ] **0.4: Refactor Existing Connection Logic**
    - **Component**: Client (`cmd/client/vpn_client.go`, `cmd/client/network_manager.go`, `cmd/client/ui_manager.go`, etc.)
    - **Details**: Move existing logic related to network connection state, computer management, and any direct signaling client interactions into the `VPNConnectionManager`. The goal is to make `vpn_client.go` and other UI-related files much leaner, primarily focusing on UI presentation and event handling, delegating all VPN core logic to the manager.
    - **Action**:
        - **Identify and Migrate State**: Review `vpn_client.go`, `network_manager.go`, and any other files that currently hold state about the network connection (e.g., `isConnected`, `currentNetwork`, `peers`). Move these fields into the `VPNConnectionManager` struct.
        - **Identify and Migrate Logic**: Review functions that:
            - Initiate network joins/leaves.
            - Process `ComputerJoined`/`ComputerLeft` notifications (even if just logging for now).
            - Manage the list of connected computers.
            - Directly send messages via the signaling client (except for the `VPNConnectionManager` itself).
            Migrate these functions or their core logic into appropriate methods within the `VPNConnectionManager`.
        - **Update Call Sites**: Change all external calls to this migrated logic to instead call the corresponding methods on the `VPNConnectionManager` instance.
        - **Simplify UI Components**: Ensure that `home_tab_component.go` and other UI-related files primarily interact with the `VPNConnectionManager` for all VPN-related operations and status updates, reducing their internal complexity.
        - **Remove Redundancy**: Delete any redundant code or fields from the original files once their responsibilities have been fully transferred to the `VPNConnectionManager`.


---

## Section 1: Core VPN Infrastructure (Client-Side)

This section focuses on creating the virtual network adapter that the operating system will use to send and receive VPN traffic.

- [ ] **1.1: Integrate a TUN/TAP Library**
    - **Component**: Client (`cmd/client/go.mod`, `cmd/client/vpn_connection_manager.go`)
    - **Details**: The core of the VPN is a virtual network interface. We need to add a library to create and manage this interface in a cross-platform way. This task involves adding the necessary dependency and making it available to the `VPNConnectionManager`.
    - **Action**:
        - **1.1.1: Add `water` dependency**: In `cmd/client/go.mod`, add `github.com/songgao/water` as a direct dependency. Run `go mod tidy` to ensure dependencies are resolved.
        - **1.1.2: Update `VPNConnectionManager` struct**: In `cmd/client/vpn_connection_manager.go`, ensure the `VPNConnectionManager` struct includes a field for the TUN interface, e.g., `tunInterface *water.Interface`.
    - **Note**: This operation requires administrator/root privileges to succeed. The application must be run with elevated permissions.

- [ ] **1.2: Implement TUN Interface Lifecycle**
    - **Component**: Client (`cmd/client/vpn_connection_manager.go`, `libs/network/network.go` - if `network.go` is used for OS-specific commands)
    - **Details**: The TUN interface should be created when a computer successfully connects to a network and destroyed when they disconnect. This involves OS-specific commands for configuration.
    - **Action**:
        - **1.2.1: Create TUN Interface**: In `VPNConnectionManager`, implement a method (e.g., `createTUNInterface()`) that:
            - Calls `water.New(water.Config{DeviceType: water.TUN})` to create the TUN device.
            - Stores the returned `*water.Interface` in the `tunInterface` field of `VPNConnectionManager`.
            - Handles potential errors during interface creation.
        - **1.2.2: Configure TUN Interface with IP**: After receiving the `NetworkJoined` message (which includes the `assignedIP` from the server, see 4.1), call OS-specific commands to configure the TUN interface with this IP and its subnet mask (e.g., `10.10.0.5/24`). This might involve using `os/exec` to run `ip addr add` (Linux), `ifconfig` (macOS), or `netsh interface ip set address` (Windows).
        - **1.2.3: Bring TUN Interface Up**: Use OS-specific commands to bring the interface up (e.g., `ip link set dev tun0 up` on Linux, `ifconfig tun0 up` on macOS).
        - **1.2.4: Implement TUN Interface Teardown**: Implement a method (e.g., `closeTUNInterface()`) in `VPNConnectionManager` that:
            - Closes the `water.Interface` (e.g., `v.tunInterface.Close()`).
            - Calls OS-specific commands to bring the interface down and remove its IP configuration (e.g., `ip link set dev tun0 down`, `ip addr del`).
        - **1.2.5: Integrate Lifecycle with Connection Flow**: Call `createTUNInterface()` and `closeTUNInterface()` from `ConnectToNetwork()` and `Disconnect()` methods respectively within `VPNConnectionManager` at the appropriate stages of the connection lifecycle.

---

## Section 2: P2P WebRTC Connectivity

This section details the establishment of direct WebRTC data channels between all computers in a network.

- [x] **2.1: Enhance Signaling Protocol**
    - **Component**: Server & Client (`libs/models/models.go`, `cmd/server/websocket_server.go`)
    - **Details**: The current signaling protocol needs to be extended to support WebRTC negotiation (SDP and ICE candidates).
    - **Action**:
        - Add new `MessageType` values to `models.go` for `SdpOffer`, `SdpAnswer`, and `IceCandidate`.
        - Create corresponding structs for these messages. They should include a `TargetPublicKey` field to route the message to the correct computer.
        - The server has been updated to parse these new messages and forward them to the specified `TargetPublicKey` within the same network.

- [ ] **2.2: Implement Full-Mesh ComputerConnection Logic**
    - **Component**: Client (`cmd/client/vpn_connection_manager.go`)
    - **Details**: Each client in a network must establish a direct WebRTC connection with every other client in that network. This involves initiating peer connections and handling the SDP offer/answer exchange.
    - **Action**:
        - **2.2.1: Add `pion/webrtc` dependency**: In `cmd/client/go.mod`, add `github.com/pion/webrtc/v3` as a direct dependency. Run `go mod tidy`.
        - **2.2.2: Initialize PeerConnection**: In `VPNConnectionManager`, when a `models.ComputerJoined` notification is received (via `HandleSignalingMessage`):
            - Create a new `webrtc.PeerConnection` instance for the new computer.
            - Configure ICE servers (e.g., Google's STUN server: `stun:stun.l.google.com:19302`).
            - Set up `OnICECandidate` callback to send ICE candidates to the remote peer via the signaling server (`models.MessageTypeIceCandidate`).
            - Set up `OnConnectionStateChange` callback to monitor the connection status.
            - Set up `OnDataChannel` callback to handle incoming data channels from the remote peer.
            - Create a data channel (e.g., `peerConnection.CreateDataChannel("vpn", nil)`) for sending VPN packets.
            - Store the `PeerConnection` in the `peerConnections` map using the remote computer's public key as the key.
        - **2.2.3: Create and Send SDP Offer**: For the newly created `PeerConnection`:
            - Create an SDP offer (`peerConnection.CreateOffer(nil)`).
            - Set the local description (`peerConnection.SetLocalDescription(offer)`).
            - Send the SDP offer to the new computer via the signaling server using a `models.MessageTypeSdpOffer` message. The message should include the offer's SDP and the `TargetPublicKey` of the new computer.
        - **2.2.4: Handle Incoming SDP Offer**: In `VPNConnectionManager.HandleSignalingMessage`, when a `models.MessageTypeSdpOffer` is received:
            - Create a new `webrtc.PeerConnection` (if one doesn't already exist for the sender).
            - Set the remote description (`peerConnection.SetRemoteDescription(offer)`).
            - Create an SDP answer (`peerConnection.CreateAnswer(nil)`).
            - Set the local description (`peerConnection.SetLocalDescription(answer)`).
            - Send the SDP answer back to the sender via the signaling server using a `models.MessageTypeSdpAnswer` message.
        - **2.2.5: Handle Incoming SDP Answer**: In `VPNConnectionManager.HandleSignalingMessage`, when a `models.MessageTypeSdpAnswer` is received:
            - Retrieve the corresponding `PeerConnection` from the `peerConnections` map.
            - Set the remote description (`peerConnection.SetRemoteDescription(answer)`).
        - **2.2.6: Handle Incoming ICE Candidate**: In `VPNConnectionManager.HandleSignalingMessage`, when a `models.MessageTypeIceCandidate` is received:
            - Retrieve the corresponding `PeerConnection` from the `peerConnections` map.
            - Add the ICE candidate to the peer connection (`peerConnection.AddICECandidate(candidate)`).

- [ ] **2.3: Manage Computer Connections & Data Channels**
    - **Component**: Client (`cmd/client/vpn_connection_manager.go`)
    - **Details**: We need a robust way to manage the collection of active P2P connections and their associated data channels. This includes handling connection closures and ensuring data channels are ready for packet routing.
    - **Action**:
        - **2.3.1: Store Data Channels**: In `VPNConnectionManager`, modify the `peerConnections` map to also store a reference to the `webrtc.DataChannel` once it's established and ready. A nested map or a custom struct might be useful (e.g., `map[string]struct{ PeerConnection *webrtc.PeerConnection; DataChannel *webrtc.DataChannel }`).
        - **2.3.2: Handle Data Channel `OnOpen`**: In the `OnDataChannel` callback (set in 2.2.2) and for the data channel created in 2.2.2, implement the `OnOpen` event handler. When `OnOpen` fires, it means the data channel is ready for use. Log this event and update the internal state to reflect the data channel's readiness.
        - **2.3.3: Handle `ComputerLeft` Notification**: In `VPNConnectionManager.HandleSignalingMessage`, when a `models.MessageTypeComputerLeft` notification is received:
            - Retrieve the corresponding `PeerConnection` from the `peerConnections` map.
            - Close the `PeerConnection` (`peerConnection.Close()`). This will also close associated data channels.
            - Remove the `PeerConnection` (and its associated data channel) from the `peerConnections` map.
        - **2.3.4: Handle `OnConnectionStateChange`**: In the `OnConnectionStateChange` callback (set in 2.2.2), monitor the state transitions of the `PeerConnection`. When the state changes to `webrtc.PeerConnectionStateDisconnected` or `webrtc.PeerConnectionStateFailed`, clean up the associated resources (close the connection and remove from map) similar to `ComputerLeft` handling.
        - **2.3.5: Implement Data Channel `OnClose` and `OnError`**: For each data channel, implement `OnClose` and `OnError` callbacks to handle graceful closures or unexpected errors, logging them and potentially triggering cleanup of the associated peer connection.

---

## Section 3: Packet Routing

This is the heart of the VPN: reading real IP packets, sending them to the correct computer, and writing received packets back to the system.

- [ ] **3.1: Implement Packet Forwarding: TUN -> WebRTC**
    - **Component**: Client (`cmd/client/vpn_connection_manager.go`)
    - **Details**: Capture outgoing IP packets from the OS (via the TUN interface) and forward them to the correct computer over WebRTC. This involves reading from the TUN device, parsing the IP header to determine the destination, and sending the packet via the appropriate WebRTC DataChannel.
    - **Action**:
        - **3.1.1: Start TUN Read Goroutine**: In `VPNConnectionManager`, when the TUN interface is successfully created and brought up (after 1.2.5), start a dedicated goroutine (e.g., `go v.readTUNPackets()`).
        - **3.1.2: Read Packets from TUN**: Inside `readTUNPackets()`, implement a loop that continuously reads raw IP packets from `v.tunInterface.Read(packetBuffer)`. Use a sufficiently large `packetBuffer` (e.g., `65535` bytes for maximum IP packet size).
        - **3.1.3: Parse Destination IP**: For each `packetBuffer` read, parse the IP header to extract the destination IP address. Go's `net` package can help with this (e.g., `ip.Header.Dst`).
        - **3.1.4: Route Packet to Peer**: Look up the destination IP in a routing table or a mapping (which you'll need to maintain in `VPNConnectionManager`, mapping virtual IPs to computer public keys/DataChannels). This mapping will tell you which `webrtc.DataChannel` to use.
        - **3.1.5: Send Packet over DataChannel**: Retrieve the correct computer's `DataChannel` from the `peerConnections` map (or a similar structure). Send the raw `packetBuffer` bytes directly over this Data Channel using `dataChannel.Send(packetBuffer[:n])`, where `n` is the number of bytes read.
        - **3.1.6: Handle Errors**: Implement error handling for `ifce.Read()` and `dataChannel.Send()`, logging any issues.

- [ ] **3.2: Implement Packet Forwarding: WebRTC -> TUN**
    - **Component**: Client (`cmd/client/vpn_connection_manager.go`)
    - **Details**: Receive IP packets from other computers over WebRTC DataChannels and write them to the TUN interface so the local OS can process them. This completes the packet flow for incoming VPN traffic.
    - **Action**:
        - **3.2.1: Implement DataChannel `OnMessage` Handler**: In `VPNConnectionManager`, when setting up each `webrtc.DataChannel` (in 2.2.2 or 2.3.1), attach an `OnMessage` handler (e.g., `dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) { v.handleDataChannelMessage(msg) })`).
        - **3.2.2: Write Packet to TUN**: Inside `handleDataChannelMessage(msg webrtc.DataChannelMessage)`, check if `msg.IsBinary` is true (as VPN packets are binary). Take the received byte slice (`msg.Data`) and write it directly to the TUN interface: `v.tunInterface.Write(msg.Data)`.
        - **3.2.3: Handle Errors**: Implement error handling for `v.tunInterface.Write()`, logging any issues.
        - **3.2.4: Ensure DataChannel Readiness**: This task relies on the DataChannels being properly established and ready (as handled in 2.3). Ensure that packets are only written to the TUN interface once the DataChannel is in an `Open` state.

---

## Section 4: Network & IP Management

This section covers the logistics of IP address assignment and system routing configuration.

- [x] **4.1: Implement IP Address Management (IPAM)**
    - **Component**: Server
    - **Details**: The server needs to act as a simple IPAM service to ensure no two clients in a network have the same virtual IP.
    - **Action**:
        - When a network is created, associate an IP subnet with it (e.g., `10.10.0.0/24`).
        - When a computer joins a network, assign them the next available IP from that subnet (e.g., `10.10.0.X`).
        - Send this assigned IP to the client in the `NetworkJoined` response.
        - When a computer leaves, release their IP back to the pool for that network.

- [ ] **4.2: Configure System Routing Table**
    - **Component**: Client (`cmd/client/vpn_connection_manager.go`)
    - **Details**: For the OS to know which traffic should go through the VPN, a route must be added to its routing table. This is a critical step to ensure that traffic destined for the VPN network is correctly routed through the TUN interface.
    - **Action**:
        - **4.2.1: Determine OS-Specific Commands**: Identify the appropriate shell commands for adding and deleting routes on Linux, macOS, and Windows. These commands typically involve `ip route` (Linux), `route add` (macOS/Windows), or `netsh interface ip add route` (Windows).
        - **4.2.2: Implement Route Addition**: In `VPNConnectionManager`, after the TUN interface is successfully created and configured with its IP (in task 1.2.5), implement a method (e.g., `addVPNRoute()`) that:
            - Constructs the correct OS-specific command to add a route for the entire VPN subnet (e.g., `10.10.0.0/24`) through the TUN interface.
            - Executes this command using `os/exec.Command()`. This will require elevated privileges.
            - Handles potential errors during command execution.
        - **4.2.3: Implement Route Deletion**: Implement a corresponding method (e.g., `deleteVPNRoute()`) in `VPNConnectionManager` that:
            - Constructs the correct OS-specific command to delete the previously added route.
            - Executes this command using `os/exec.Command()`. This will also require elevated privileges.
            - Handles potential errors.
        - **4.2.4: Integrate Route Management with Connection Flow**: Call `addVPNRoute()` from `ConnectToNetwork()` after the TUN interface is up and configured, and call `deleteVPNRoute()` from `Disconnect()` before closing the TUN interface. Ensure proper error handling and logging for these operations.
    - **Note**: This is another action that requires administrator/root privileges. The command will differ between Linux, macOS, and Windows.