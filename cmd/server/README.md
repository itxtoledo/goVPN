# GoVPN Server

Signaling server for the GoVPN application, built in Go. This server manages communication between clients, allowing them to establish secure and efficient P2P connections.

## Architecture

The GoVPN server follows a simple architecture, focused on signaling, where the main objective is to facilitate initial communication between clients and manage rooms. There is no TLS implementation.

### Main Components

1. **WebSocketServer**: Central component responsible for:
   - Managing WebSocket connections with clients
   - Routing messages between clients
   - Administering rooms and their participants
   - Implementing basic authentication logic
   - Handling disconnection situations
   - Monitoring usage statistics
   - Managing the lifecycle of rooms

2. **SupabaseManager**: Interface for the Supabase database:
   - Persistent storage of room information
   - Querying and modifying room data
   - Managing room lifecycle (expiration, cleanup)
   - Public key ownership verification

3. **StatsManager**: Monitoring component:
   - Tracking active connections
   - Room and message statistics
   - Performance metrics
   - Endpoint for monitoring

### Main Data Structures

1. **In-Memory Mappings**:
   - `clients`: Maps WebSocket connections to room IDs
   - `networks`: Maps room IDs to lists of connections
   - `clientToPublicKey`: Associates each connection with its public key

2. **ServerRoom**: Extension of the basic room structure:
   - Fundamental room data (ID, name, password)
   - Owner's public key
   - Metadata (creation, last activity)

## Operation Flow

```
WebSocket Client → WebSocketServer → [Message Processing]
                                       ↓
                                   [Validation]
                                       ↓
                            [Specific Action by Type]
                           /       |        \        \
                     Create Room  Join    Leave    Other Actions
                         |         |         |          |
                         v         v         v          v
                     [Supabase] [Notify] [Cleanup] [Process]
                                  Others    Resources
```

## Room Management

1. **Room Creation**:
   - Data validation (password, name)
   - Unique ID generation
   - Persistence in Supabase
   - Owner association
   - Public key uniqueness verification

2. **Room Joining**:
   - Credential validation
   - Limit verification
   - Notification to existing peers
   - Connection tracking
   - Room activity update

3. **Room Leaving**:
   - Client removal from the room
   - Conditional cleanup based on ownership
   - Notification to other participants
   - Optional room preservation

4. **Room Ownership**:
   - Public key as owner identifier
   - Special permissions (rename, kick)
   - Automatic deletion when owner leaves (if not configured to preserve)

## Messaging System

The server implements a JSON-over-WebSocket messaging protocol:

- **SignalingMessage**: Envelope structure for all messages
- **Message Types**: CreateRoom, JoinRoom, LeaveRoom, Ping, Rename, Kick, etc.
- **Identification**: Each message has a unique ID for tracking and response correlation

Full API details can be found in `docs/websocket_api.md`.

## Security Features

- **Public Key Verification**: Identity validation via Ed25519 keys
- **Room Authentication**: Password protection for room access
- **Room Isolation**: Messages are routed only within the correct rooms
- **Data Validation**: Strict verification of user inputs
- **Access Control**: Only owners can perform administrative actions
- **Timeouts**: Automatic disconnection of inactive clients

## Monitoring and Metrics

The server provides a `/stats` endpoint that returns real-time metrics:

- Total number of connections
- Active connections
- Processed messages
- Active rooms
- Cleanup statistics
- Uptime

## Technologies Used

- **Go**: Main programming language (Go 1.18+)
- **Gorilla WebSocket**: Library for WebSocket connection management
- **Supabase-Go**: Supabase client for Go
- **Ed25519**: For signature verification and authentication

## Data Persistence

Room data is stored in Supabase with the following fields:
- Room ID
- Room Name
- Password (hash)
- Owner's public key
- Creation timestamp
- Last activity timestamp

## Performance Characteristics

- **Efficient Memory Usage**: Optimized data structures
- **Concurrency**: Leveraging goroutines for parallel operations
- **Automatic Cleanup**: Scheduled removal of inactive rooms to free up resources
- **Graceful Shutdown**: Notification to clients and state persistence during restarts
- **Timeouts**: Prevention of resource leaks from pending connections

## Configuration

The server is configured via environment variables:

```bash
# Required
export SUPABASE_URL="your-supabase-url"
export SUPABASE_KEY="your-supabase-key"

# Optional
export PORT="8080"
export MAX_CLIENTS_PER_ROOM="50"
export ROOM_EXPIRY_DAYS="7"
export LOG_LEVEL="info"
export READ_BUFFER_SIZE="4096"
export WRITE_BUFFER_SIZE="4096"
export CLEANUP_INTERVAL="24h"
export SUPABASE_ROOMS_TABLE="govpn_rooms"
export ALLOW_ALL_ORIGINS="true"
```

## Endpoints

- `/ws`: Main endpoint for WebSocket connections
- `/health`: Server health check (returns status 200 if operational)
- `/stats`: Returns real-time server statistics in JSON format

## Running the Server

```bash
cd cmd/server && go run .
```

## Graceful Shutdown

The server supports graceful shutdown, where:

1. All connected clients are notified of the imminent shutdown
2. The current state of the rooms is preserved in Supabase
3. Connections are closed orderly
4. Resources are released before termination

## Limitations

- Does not directly implement TLS (recommended to use behind a proxy like Nginx or Traefik)
- Scales vertically, not horizontally
- No clustered database (uses only Supabase)
- No integrated load balancing

## Main Dependencies

- github.com/gorilla/websocket
- github.com/supabase-community/supabase-go
- crypto/ed25519
