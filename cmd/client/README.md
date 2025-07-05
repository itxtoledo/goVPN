# GoVPN Client

GoVPN application client built in Go with a graphical interface using the Fyne v2 library.

## Architecture

The GoVPN client follows a modular architecture with the following main components:

### Core Components

1. **VPNClient**: Central component that coordinates all other client components.
   - Manages the application lifecycle
   - Integrates all other components

2. **NetworkManager**: Responsible for managing network connections.
   - Establishes connections with the signaling server
   - Manages room creation and joining
   - Coordinates P2P connection with other clients

3. **SignalingClient**: Manages WebSocket communication with the server.
   - Sends and receives signaling messages
   - Processes event notifications (new peers, peer departures)
   - Implements the communication protocol defined in `models`

### Data Storage

1. **DatabaseManager**: Manages the local SQLite database.
   - Stores information about saved rooms
   - Keeps a record of previous connections
   - Stores cryptographic keys

2. **ConfigManager**: Manages user settings.
   - Stores preferences like language and theme
   - Handles server address and other configurations

3. **RealtimeDataLayer**: Real-time data layer for the interface.
   - Provides data bindings for Fyne widgets
   - Implements the Observer pattern for change notification
   - Centralizes application state

### User Interface

1. **UIManager**: Manages the graphical user interface.
   - Coordinates navigation between screens
   - Integrates UI components
   - Manages the UI lifecycle

2. **UI Components**:
   - **HeaderComponent**: Displays the header with connection status
   - **HomeTabComponent**: Main screen with room list and options
   - **SettingsTabComponent**: Application settings
   - **NetworkListComponent**: List of available rooms
   - **RoomItemComponent**: Visual representation of a room

3. **Dialogs**:
   - **ConnectDialog**: Dialog to connect to a room
   - **RoomDialog**: Dialog to create/join rooms

## Data Flow

```
UI Events → UIManager → NetworkManager → SignalingClient → WebSocket → Server
    ↑                      ↓                                   ↑
    └──── RealtimeData ← Events                                |
                                                              ↓
            Data Storage ← DatabaseManager ←→ ConfigManager
```

## Technologies Used

- **Go**: Main programming language (Go 1.18+)
- **Fyne v2**: Cross-platform UI framework
  - Native widgets for various platforms
  - Responsive layout system
  - Light/dark theme support

- **SQLite**: Local database
  - Storage of configurations and persistent data
  - Via `database/sql` package and SQLite driver

- **WebSocket**: Real-time communication
  - Persistent connection with the signaling server
  - Implemented via the Gorilla WebSocket library

- **Ed25519**: Public key cryptography
  - Secure message authentication
  - Key generation and storage

## File Structure

- **main.go**: Application entry point
- **about_window.go**: About window implementation
- **base_window.go**: Base window structure
- **config_manager.go**: Configuration management
- **header_component.go**: Header UI component
- **home_tab_component.go**: Home tab UI component
- **network_list_component.go**: Network list UI component
- **network_manager.go**: Network connection management
- **password_validator.go**: Password validation logic
- **room_item_component.go**: Room item UI component
- **settings_tab_component.go**: Settings tab UI component
- **signaling_client.go**: WebSocket signaling client
- **ui_manager.go**: User interface management
- **vpn_client.go**: VPN logic coordination
- **data/**: Real-time data layer components
  - **realtime_data.go**: Real-time data implementation
- **dialogs/**: UI dialog components
  - **connect_dialog.go**: Connect dialog
  - **dialogs_factory.go**: Dialogs factory
  - **dialogs.go**: Dialogs main file
  - **join_dialog.go**: Join dialog
  - **room_dialog.go**: Room dialog
- **icon/**: Visual resources and icons
  - **icon.go**: Icon definitions
  - **assets/**: Icon assets (e.g., `app.png`, `link_off.svg`)
- **storage/**: Persistent storage management
  - **config.go**: Configuration storage
  - **database_manager.go**: Database management

## Important Features

- **Fixed size**: 300x600 pixels for cross-platform compatibility
- **Responsive design**: Adaptive layout within fixed dimensions
- **Local storage**: All data persisted only locally in SQLite
- **Secure communication**: Public key-based authentication
- **Real-time updates**: Reactive interface using Fyne bindings

## Running the Client

```bash
cd cmd/client && go run .
```

## Main Dependencies

- fyne.io/fyne/v2
- github.com/mattn/go-sqlite3
- github.com/gorilla/websocket
- crypto/ed25519