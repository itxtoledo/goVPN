module github.com/itxtoledo/govpn/libs/signaling

go 1.22.0

require (
	github.com/gorilla/websocket v1.5.3
	github.com/itxtoledo/govpn/libs/utils v0.0.0-00010101000000-000000000000
	github.com/itxtoledo/govpn/libs/signaling/models v0.0.0-00010101000000-000000000000
)

replace github.com/itxtoledo/govpn/libs/utils v0.0.0-00010101000000-000000000000 => ../utils

replace github.com/itxtoledo/govpn/libs/signaling/models v0.0.0-00010101000000-000000000000 => ./models
