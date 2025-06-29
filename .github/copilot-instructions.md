This is a VPN project made with golang.
The client must be 300x600 fixed size.
Use only available fyne v2.4 widgets.
To update texts and itens on widgets use fyne bindings.
The server is just a websocket relayer to receive messages from clients.
The client store things only on sqlite using the database_manager and config.
All data must be sent to server using available structs on models package.
Do not implement TLS!
This project is located at github.com/itxtoledo/govpn
To run server: cd cmd/server && go run .
To run client: cd cmd/client && go run .
DO NOT USE fyne.CurrentApp().Driver().RunOnMain because this method dont exist on new Fyne (v2.4+)!