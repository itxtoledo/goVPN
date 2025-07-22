module github.com/itxtoledo/govpn/libs/signaling/client

go 1.22.0

require (
	github.com/gorilla/websocket v1.5.3
	github.com/itxtoledo/govpn/libs/signaling/models v0.0.0
	github.com/itxtoledo/govpn/libs/utils v0.0.0
)

replace (
	github.com/itxtoledo/govpn/libs/signaling/models v0.0.0 => ../models
	github.com/itxtoledo/govpn/libs/utils v0.0.0 => ../../utils
)
