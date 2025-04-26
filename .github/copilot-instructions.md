This is a VPN project made with golang.
The client must be 300x600 fixed size.
Use only available fyne 2.0 widgets.
To update texts and itens on widgets use fyne bindings.
The server is just a websocket relayer to receive messages from clients.
The client store things only on sqlite using the database_manager and config.
All data must be sent to server using available structs on models package.