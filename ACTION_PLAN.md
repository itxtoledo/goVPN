# goVPN - Full VPN Implementation Plan

This document outlines the necessary steps to evolve goVPN from a signaling/discovery application into a full computer-to-computer VPN client, similar to Hamachi, using WebRTC and a virtual network interface (TUN).

---

## Section 1: Core VPN Infrastructure (Client-Side)

This section focuses on creating the virtual network adapter that the operating system will use to send and receive VPN traffic.

- [ ] **1.1: Integrate a TUN/TAP Library**
    - **Component**: Client
    - **Details**: The core of the VPN is a virtual network interface. We need to add a library to create and manage this interface in a cross-platform way.
    - **Action**: Add the `github.com/songgao/water` dependency to the client's `go.mod`. This library allows creating TUN interfaces on Windows, macOS, and Linux.
    - **Note**: This operation requires administrator/root privileges to succeed. The application must be run with elevated permissions.

- [ ] **1.2: Implement TUN Interface Lifecycle**
    - **Component**: Client (`cmd/client/vpn_client.go`, `libs/network/network.go`)
    - **Details**: The TUN interface should be created when a computer connects to a network and destroyed when they disconnect.
    - **Action**:
        - Create a function `createTUNInterface()` that initializes the interface using `water.New()`.
        - Upon connecting to a network and receiving a virtual IP from the server (see 4.1), configure the interface with this IP and its subnet mask (e.g., `10.10.0.5/24`).
        - Bring the interface up using OS-specific commands (`ip addr`, `ifconfig`).
        - Implement a `closeTUNInterface()` function to be called on disconnect.

---

## Section 2: P2P WebRTC Connectivity

This section details the establishment of direct WebRTC data channels between all computers in a network.

- [ ] **2.1: Enhance Signaling Protocol**
    - **Component**: Server & Client (`libs/models/models.go`, `cmd/server/websocket_server.go`)
    - **Details**: The current signaling protocol needs to be extended to support WebRTC negotiation (SDP and ICE candidates).
    - **Action**:
        - Add new `MessageType` values to `models.go` for `SdpOffer`, `SdpAnswer`, and `IceCandidate`.
        - Create corresponding structs for these messages. They should include a `TargetID` field to route the message to the correct computer.
        - The server must be updated to parse these new messages and forward them to the specified `TargetID` within the same network.

- [ ] **2.2: Implement Full-Mesh ComputerConnection Logic**
    - **Component**: Client
    - **Details**: Each client in a network must establish a direct WebRTC connection with every other client in that network.
    - **Action**:
        - When a `ComputerJoined` notification is received, the existing client should initiate a new `webrtc.ComputerConnection` for the new computer.
        - Create an SDP offer and send it to the new computer via the signaling server (`SdpOffer` message).
        - Implement handlers to process `SdpOffer`, `SdpAnswer`, and `IceCandidate` messages from other computers to complete the connection lifecycle.
        - Use the `github.com/pion/webrtc/v3` library, which you likely already have as a dependency through Fyne.

- [ ] **2.3: Manage Computer Connections & Data Channels**
    - **Component**: Client
    - **Details**: We need a way to manage the collection of active P2P connections.
    - **Action**:
        - Create a map to store active connections, for instance: `connections := make(map[string]*webrtc.DataChannel)`. The key should be the computer's public key.
        - When a `ComputerLeft` notification is received, find the corresponding `ComputerConnection` in the map, close it, and remove it.
        - When a Data Channel's `OnOpen` event fires, it is ready for routing packets.

---

## Section 3: Packet Routing

This is the heart of the VPN: reading real IP packets, sending them to the correct computer, and writing received packets back to the system.

- [ ] **3.1: Implement Packet Forwarding: TUN -> WebRTC**
    - **Component**: Client
    - **Details**: Capture outgoing IP packets from the OS (via the TUN interface) and forward them to the correct computer over WebRTC.
    - **Action**:
        - Create a dedicated goroutine that runs in a loop: `ifce.Read(packetBuffer)`.
        - For each packet read, parse its header to determine the destination IP address.
        - Look up the destination IP in a routing table (which maps virtual IPs to computer public keys) to find the target computer.
        - Retrieve the correct computer's `DataChannel` from the map created in 2.3.
        - Send the raw `packetBuffer` bytes directly over the Data Channel.

- [ ] **3.2: Implement Packet Forwarding: WebRTC -> TUN**
    - **Component**: Client
    - **Details**: Receive IP packets from computers over WebRTC and write them to the TUN interface so the OS can process them.
    - **Action**:
        - The `OnMessage` handler for each Data Channel will receive raw IP packet data.
        - Take the received byte slice (`message.Data`) and write it directly to the TUN interface: `ifce.Write(message.Data)`.
        - The OS will handle the rest, delivering the packet to the appropriate application.

---

## Section 4: Network & IP Management

This section covers the logistics of IP address assignment and system routing configuration.

- [ ] **4.1: Implement IP Address Management (IPAM)**
    - **Component**: Server
    - **Details**: The server needs to act as a simple IPAM service to ensure no two clients in a network have the same virtual IP.
    - **Action**:
        - When a network is created, associate an IP subnet with it (e.g., `10.20.30.0/24`).
        - When a computer joins a network, assign them the next available IP from that subnet.
        - Send this assigned IP to the client in the `NetworkJoined` response.
        - When a computer leaves, release their IP back to the pool for that network.

- [ ] **4.2: Configure System Routing Table**
    - **Component**: Client
    - **Details**: For the OS to know which traffic should go through the VPN, a route must be added to its routing table.
    - **Action**:
        - After the TUN interface is created and configured with its IP (e.g., `10.20.30.5`), use `os/exec` to run a command that adds a route for the entire VPN subnet.
        - **Example (Linux)**: `sudo ip route add 10.20.30.0/24 dev govpn0`
        - **Note**: This is another action that requires administrator/root privileges. The command will differ between Linux, macOS, and Windows.
