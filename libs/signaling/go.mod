module github.com/itxtoledo/govpn/libs/signaling

go 1.22

require (
	github.com/gorilla/websocket v1.5.3
	github.com/itxtoledo/govpn/libs/models v0.0.0-00010101000000-000000000000
)

replace github.com/itxtoledo/govpn/libs/models v0.0.0-00010101000000-000000000000 => ../models
