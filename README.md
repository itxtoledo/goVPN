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