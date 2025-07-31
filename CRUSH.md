# CRUSH Context for GoVPN

## Project Overview
- VPN project built with Go
- Client uses Fyne v2.4 with fixed 300x600 size
- Server is a WebSocket relay for client messages
- P2P communication with WebRTC for actual data transfer

## Build Commands
```bash
# Server
cd cmd/server && go build .

# Client
cd cmd/client && go build .

# Run server (development)
cd cmd/server && go run main.go

# Run client (development)
cd cmd/client && go run .
```

## Test Commands
```bash
# Run all tests
go test ./...

# Run a specific test file
go test ./path/to/package -v

# Run tests with coverage
go test ./... -cover
```

## Lint/Format Commands
```bash
# Format code
go fmt ./...

# Vet code
go vet ./...
```

## Code Style Guidelines

### Imports
- Use standard library imports first
- Separate third-party and local imports with blank line
- Group imports logically

### Formatting
- Use `go fmt` for all formatting
- Max line length 100 characters
- Use tabs for indentation

### Types/Naming
- Use camelCase for variables/functions
- Use PascalCase for exported types/functions
- Use descriptive names even if longer

### Error Handling
- Always handle errors explicitly
- Use `errors.New` or `fmt.Errorf` for error creation
- Wrap errors with context when passing up the stack

### Fyne Specific
- Client must be 300x600 fixed size
- Use only available Fyne v2.4 widgets
- Update texts/items on widgets using Fyne bindings
- DO NOT use `fyne.CurrentApp().Driver().RunOnMain`

### Architecture
- Client stores data only in config
- All data sent to server using structs from models package
- Server is just a WebSocket relay (no TLS)
- Use existing libraries and utilities in the codebase