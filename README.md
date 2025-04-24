# GoVPN

A Virtual LAN (VLAN) solution for gaming that allows players to connect as if they were on the same local network.

## Features

- Create and join virtual game rooms
- NAT traversal using STUN/TURN for P2P connections
- End-to-end encryption
- Cross-platform support (Windows, macOS, Linux)
- Local data storage using SQLite

## Arquitetura do Sistema

O GoVPN é organizado em uma arquitetura cliente-servidor modular, com comunicação P2P entre os clientes para minimizar a latência. A arquitetura está estruturada da seguinte forma:

### Bibliotecas Compartilhadas

- **libs/crypto_utils**: Implementa funções criptográficas para garantir a segurança da comunicação, incluindo:
  - Geração de chaves RSA
  - Assinatura e verificação de mensagens
  - Criptografia AES-GCM para os dados transmitidos
  - Geração de identificadores seguros

- **libs/models**: Define as estruturas de dados compartilhadas entre cliente e servidor:
  - Room: Representa uma sala de jogo virtual
  - Message: Define o formato das mensagens trocadas via WebSocket
  - NetworkPacket: Estrutura para pacotes de rede tunelados pela VPN
  - ClientInfo: Informações sobre os clientes conectados

- **libs/network**: Gerencia a rede virtual entre os clientes:
  - VirtualNetwork: Classe principal que coordena a comunicação entre peers
  - Mapeamento de endereços IP virtuais
  - Encapsulamento e roteamento de pacotes entre clientes

### Serviços

- **services/server**: Servidor de sinalização que facilita:
  - Criação e gerenciamento de salas
  - Autenticação das operações via assinaturas RSA
  - Estabelecimento de conexões WebRTC entre clientes
  - Persistência de dados em Supabase

- **services/client**: Aplicação cliente com interface gráfica que permite:
  - Criação e entrada em salas de jogo
  - Gerenciamento da conexão P2P via WebRTC
  - Armazenamento local de configurações em SQLite
  - Interface gráfica construída com Fyne

### Fluxo de Comunicação

1. **Fase de Sinalização**:
   - O servidor atua como intermediário para estabelecer conexões iniciais
   - Clientes se autenticam e trocam informações sobre salas disponíveis
   
2. **Estabelecimento de Conexão P2P**:
   - Troca de ofertas WebRTC, respostas e candidatos ICE via servidor
   - Uso de servidores STUN para descoberta de endereços públicos
   - Fallback para servidores TURN quando a conexão direta falha

3. **Comunicação Direta**:
   - Após estabelecida, a comunicação ocorre diretamente entre os clientes
   - Dados são criptografados ponta a ponta (chave derivada da senha da sala)

4. **Rede Virtual**:
   - Cada cliente recebe um endereço IP virtual (formato 10.0.0.x)
   - Pacotes de rede são encapsulados, criptografados e enviados pelo canal de dados WebRTC

## Project Structure

GoVPN is organized into a library-and-services structure for better maintainability:

- **libs/**: Contains shared libraries used by both client and server components
  - **libs/crypto_utils/**: Cryptographic utilities for secure communication
  - **libs/models/**: Shared data structures used throughout the application
  - **libs/network/**: Network management functionality for virtual networking

- **services/**: Contains the main applications
  - **services/server/**: Server implementation for handling signaling and room management
  - **services/client/**: Client application with GUI for creating and joining game networks

This organization allows the server and client components to be built, deployed, and versioned independently while sharing common functionality through the libraries.

## Components

- **Server**: Signaling server for connection establishment
- **Client**: GUI application for creating and joining game networks

## Server Environment Variables

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
# Make sure you're in the project root directory
cd cmd/client
fyne package -os windows -icon ../../icon.png -name GoVPN
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
- Select either "Run GoVPN Server" or "Run GoVPN Client"

## License

[MIT License](LICENSE)