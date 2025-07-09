# GoVPN

**🚧 WORK IN PROGRESS 🚧**

This project is currently under active development. It is not yet functional, and there are no stable releases available.

A Virtual Local Area Network (VLAN) solution for games that allows players to connect as if they were on the same local network.

## Features

- Create and join virtual game networks
- NAT traversal using STUN/TURN for P2P connections
- End-to-end encryption
- Cross-platform support (Windows, macOS, Linux)
- Local data storage using SQLite

## System Architecture

GoVPN is organized in a modular client-server architecture, with P2P communication between clients to minimize latency. The architecture is structured as follows:

### Shared Libraries

- **libs/crypto_utils**: Implements cryptographic functions to ensure communication security, including:
  - Ed25519 key generation
  - Message signing and verification
  - Encryption for transmitted data
  - Secure identifier generation

- **libs/models**: Defines data structures shared between client and server:
  - Network: Represents a virtual game network
  - Message: Defines the format of messages exchanged via WebSocket
  - NetworkPacket: Structure for network packets tunneled through the VPN
  - ClientInfo: Information about connected clients

- **libs/network**: Manages the virtual network between clients:
  - VirtualNetwork: Main class that coordinates peer communication
  - Virtual IP address mapping
  - Encapsulation and routing of packets between clients

### System Components

- **cmd/server**: Signaling server that facilitates:
  - Network creation and management
  - Operation authentication via Ed25519 keys
  - Establishment of connections between clients
  - Data persistence in Supabase
  - WebSocketServer: Manages WebSocket connections with clients
  - SupabaseManager: Interface with Supabase database

- **cmd/client**: Client application with a graphical interface that allows:
  - Creating and joining game networks
  - Managing P2P connections
  - Local configuration storage in SQLite
  - Graphical interface built with Fyne (v2.0+)
  - Modular components such as NetworkManager, SignalingClient, etc.

### Communication Flow

Here's a simplified diagram of the client-server communication flow:

```mermaid
sequenceDiagram
    participant ClientA
    participant ClientB
    participant Server
    participant STUN/TURN

    ClientA->>Server: Connect (WebSocket)
    ClientB->>Server: Connect (WebSocket)

    Server-->>ClientA: Connection Acknowledged
    Server-->>ClientB: Connection Acknowledged

    ClientA->>Server: Create/Join Network Request
    Server->>ClientA: Network Confirmation / Peer List

    ClientB->>Server: Join Network Request
    Server->>ClientB: Network Confirmation / Peer List

    ClientA->>Server: Send Offer (WebRTC SDP) to ClientB
    Server-->>ClientB: Relay Offer from ClientA

    ClientB->>Server: Send Answer (WebRTC SDP) to ClientA
    Server-->>ClientA: Relay Answer from ClientB

    ClientA->>STUN/TURN: Discover Public IP
    STUN/TURN-->>ClientA: Public IP

    ClientB->>STUN/TURN: Discover Public IP
    STUN/TURN-->>ClientB: Public IP

    ClientA->>Server: Send ICE Candidates to ClientB
    Server-->>ClientB: Relay ICE Candidates from ClientA

    ClientB->>Server: Send ICE Candidates to ClientA
    Server-->>ClientA: Relay ICE Candidates from ClientB

    Note over ClientA,ClientB: P2P Connection Established
    ClientA<->>ClientB: Direct Encrypted Communication (VPN Tunnel)
```

1. **Signaling Phase**:
   - The server acts as an intermediary to establish initial connections
   - Clients authenticate and exchange information about available networks
   
2. **P2P Connection Establishment**:
   - Exchange of offers, answers, and candidates via server
   - Use of STUN servers for public address discovery
   - Fallback to TURN servers when direct connection fails

3. **Direct Communication**:
   - Once established, communication occurs directly between clients
   - Data is end-to-end encrypted (key derived from network password)

4. **Virtual Network**:
   - Each client receives a virtual IP address (format 10.0.0.x)
   - Network packets are encapsulated, encrypted, and sent through the data channel

## Current Project Structure

The current project structure is organized as follows:

```
README.md                        # Main documentation
cmd/                             # Main components
    client/                      # GoVPN client
        data/                    # Real-time data layer
        dialogs/                 # UI dialogs
        icon/                    # Icons and graphic resources
            assets/              # Image files
        storage/                 # Database and config management
        *.go                     # UI components and client logic
    server/                      # Signaling server
        docs/                    # API documentation
        *.go                     # Server implementation
libs/                            # Shared libraries
    crypto_utils/                # Cryptographic utilities
    models/                      # Data structure definitions
    network/                     # Virtual network management
migrations/                      # SQL scripts for the database
```

### Main Client Components

- **UIManager**: Manages the entire computer interface
- **VPNClient**: Controls all VPN connection logic
- **NetworkManager**: Manages network connections and networks
- **SignalingClient**: Communicates with the signaling server
- **DatabaseManager**: Manages local storage using SQLite
- **ConfigManager**: Manages application settings
- **RealtimeDataLayer**: Provides data binding to update the UI

### Main Server Components

- **WebSocketServer**: Manages WebSocket connections, networks, and clients
- **SupabaseManager**: Interface with Supabase database
- **Network Management**: Manages network creation, deletion, and modification
- **Authentication**: Authentication based on Ed25519 keys
- **Connection Management**: Manages the connection lifecycle

## Server WebSocket API

The server implements a robust WebSocket API for communication with clients. Full documentation is available at `cmd/server/docs/websocket_api.md`.

### Main Message Types

- **Client to Server**:
  - `CreateNetwork`: Creates a new network
  - `JoinNetwork`: Joins an existing network
  - `LeaveNetwork`: Leaves a network
  - `Kick`: Kicks a computer from a network
  - `Rename`: Renames a network

- **Server to Client**:
  - `NetworkCreated`: Network creation confirmation
  - `NetworkJoined`: Network join confirmation
  - `PeerJoined`: Notification of a new peer in the network
  - `PeerLeft`: Notification of a peer leaving the network
  - `NetworkDeleted`: Notification of network deletion

## Server Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Port for the server to listen on | `8080` |
| `ALLOW_ALL_ORIGINS` | Allow WebSocket connections from any origin | `true` |
| `PASSWORD_PATTERN` | Regex to validate network passwords | `^\d{4}$` |
| `MAX_NETWORKS` | Maximum number of allowed networks | `100` |
| `MAX_CLIENTS_PER_NETWORK` | Maximum number of clients in a network | `10` |
| `LOG_LEVEL` | Log level (info, debug) | `info` |
| `IDLE_TIMEOUT_SECONDS` | Timeout for inactive connections in seconds | `60` |
| `PING_INTERVAL_SECONDS` | WebSocket ping interval in seconds | `30` |
| `READ_BUFFER_SIZE` | WebSocket read buffer size | `1024` |
| `WRITE_BUFFER_SIZE` | WebSocket write buffer size | `1024` |
| `SUPABASE_URL` | Supabase URL for network persistence (required) | `""` |
| `SUPABASE_KEY` | Supabase API key for authentication (required) | `""` |
| `NETWORK_EXPIRY_DAYS` | Days after which inactive networks are deleted | `7` |
| `CLEANUP_INTERVAL_HOURS` | Interval for cleaning up expired networks in hours | `24` |

**Note:** `SUPABASE_URL` and `SUPABASE_KEY` are required for proper server operation.

## Client Interface

The GoVPN client features a graphical interface built with Fyne 2.0+ with a fixed size of 300x600 pixels. Main features:

- **Home Tab**: Displays saved networks and connection options
- **Settings Tab**: Application settings
- **Network List**: List of saved networks with connection options
- **Dialogs**: For creating/joining networks and managing connections

### Local Storage

The client stores data locally using SQLite, including:

- Computer settings
- Saved networks and passwords
- Connection history
- Cryptographic keys

## Release Process

This project uses GitHub Actions to automatically build and release the server and client components.

### Creating a Server Release

To create a new server release:

```bash
# Tag the commit with a server version
git tag server-v1.0.0
git push origin server-v1.0.0
```

This will trigger the server release workflow that builds binaries for:
- Linux (amd64)
- Windows (amd64)
- macOS (Intel/amd64)
- macOS (Apple Silicon/arm64)

### Creating a Client Release

To create a new client release:

```bash
# Tag the commit with a client version
git tag client-v1.0.0
git push origin client-v1.0.0
```

This will trigger the client release workflow that builds:
- Standalone binaries for Linux, Windows, and macOS
- Packaged applications when possible

## Manual Compilation

### Server

```bash
# Build the server executable
go build -o govpn-server ./cmd/server/main.go
```

### Client

```bash
# Build the client executable
go build -o govpn-client ./cmd/client/main.go
```

For packaged applications using Fyne:

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
# Make sure you are in the project root directory
cd cmd/client
fyne package -os windows -icon icon/assets/app.png -name GoVPN
# Or for other platforms: linux, darwin
```

## Running the Application

### Server

```bash
# Set required environment variables
export SUPABASE_URL="your-supabase-url"
export SUPABASE_KEY="your-supabase-key"

# Run the server (compiled binary)
./govpn-server
```

### Client

```bash
# Run the client (compiled binary)
./govpn-client
```

### Running from Source (Development)

To run the application directly from source code without compiling:

#### Server
```bash
# Set required environment variables
export SUPABASE_URL="your-supabase-url"
export SUPABASE_KEY="your-supabase-key"

# Run using go run (single line command)
cd cmd/server && go run main.go
```

#### Client
```bash
# Run using go run (single line command)
cd cmd/client && go run .
```

### VS Code Tasks

- Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
- Type "Tasks: Run Task"
- Select "Run GoVPN Server" or "Run GoVPN Client"

## Security Features

- Authentication based on Ed25519 keys
- Validated network passwords (default: 4 numeric digits)
- Encrypted communication between client and server
- Secure local credential persistence

## Contributions and Development

To contribute to the project:

1. Fork the repository
2. Create a branch for your feature (`git checkout -b feature/new-feature`)
3. Commit your changes (`git commit -am 'Add new feature'`)
4. Push to the branch (`git push origin feature/new-feature`)
5. Create a Pull Request

## Implementation Notes

- The client is built with Fyne 2.0+ and has a fixed size of 300x600
- The server uses Gorilla WebSocket for real-time communication
- The system uses Supabase for server data persistence
- P2P communication uses WebRTC to establish direct connections between clients

## Troubleshooting

- **Connection error**: Check if the server is running and environment variables are set
- **Fyne compilation issues**: Make sure Fyne requirements are installed (gcc, graphic dependencies)
- **SQLite errors**: Check permissions for the ~/.govpn directory

## License

[MIT License](LICENSE)