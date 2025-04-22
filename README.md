# GoVPN

A Virtual LAN (VLAN) solution for gaming that allows players to connect as if they were on the same local network.

## Features

- Create and join virtual game rooms
- NAT traversal using STUN/TURN for P2P connections
- End-to-end encryption
- Cross-platform support (Windows, macOS, Linux)
- Local data storage using SQLite

## Components

- **Server**: Signaling server for connection establishment
- **Client**: GUI application for creating and joining game networks

## Server Environment Variables

The server component can be configured using the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Port number for the server to listen on | `8080` |
| `ALLOW_ALL_ORIGINS` | Allow WebSocket connections from any origin | `true` |
| `PASSWORD_PATTERN` | Regular expression for validating room passwords | `^\d{4}$` |
| `MAX_ROOMS` | Maximum number of rooms allowed | `100` |
| `MAX_CLIENTS_PER_ROOM` | Maximum number of clients allowed in a room | `10` |
| `LOG_LEVEL` | Level of logging (info, debug) | `info` |
| `IDLE_TIMEOUT_SECONDS` | Timeout for inactive connections in seconds | `60` |
| `PING_INTERVAL_SECONDS` | WebSocket ping interval in seconds | `30` |
| `READ_BUFFER_SIZE` | WebSocket read buffer size | `1024` |
| `WRITE_BUFFER_SIZE` | WebSocket write buffer size | `1024` |
| `SUPABASE_URL` | Supabase URL for room persistence (required) | `""` |
| `SUPABASE_KEY` | Supabase API key for authentication (required) | `""` |
| `SUPABASE_ROOMS_TABLE` | Supabase table name for storing rooms | `rooms` |

**Note:** `SUPABASE_URL` and `SUPABASE_KEY` are required for the server to function properly.

## Release Process

This project uses GitHub Actions to automatically build and release both the server and client components.

### Creating a Server Release

To create a new server release:

```bash
# Tag the commit with a server version
git tag server-v1.0.0
git push origin server-v1.0.0
```

This will trigger the server release workflow which builds binaries for:
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

This will trigger the client release workflow which builds:
- Standalone binaries for Linux, Windows, and macOS
- Packaged applications where possible

## Building Manually

### Server

```bash
go build -o govpn-server ./server.go
```

### Client

```bash
go build -o govpn-client ./client.go
```

For packaged applications using Fyne:

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
fyne package -os windows -icon icon.png -name GoVPN
# Or for other platforms: linux, darwin
```

## License

[MIT License](LICENSE)