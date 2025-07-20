module github.com/itxtoledo/govpn/cmd/server

go 1.23.0

toolchain go1.23.8

require (
	github.com/gorilla/websocket v1.5.3
	github.com/itxtoledo/govpn/libs/crypto_utils v0.0.0
	github.com/itxtoledo/govpn/libs/models v0.0.0
	github.com/itxtoledo/govpn/libs/signaling v0.0.0
	github.com/joho/godotenv v1.5.1
	github.com/supabase-community/supabase-go v0.0.4
	go.uber.org/zap v1.27.0
)

replace (
	github.com/itxtoledo/govpn/libs/crypto_utils v0.0.0 => ../../libs/crypto_utils
	github.com/itxtoledo/govpn/libs/models v0.0.0 => ../../libs/models
	github.com/itxtoledo/govpn/libs/network v0.0.0 => ../../libs/network
	github.com/itxtoledo/govpn/libs/signaling v0.0.0 => ../../libs/signaling
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/supabase-community/functions-go v0.0.0-20220927045802-22373e6cb51d // indirect
	github.com/supabase-community/gotrue-go v1.2.0 // indirect
	github.com/supabase-community/postgrest-go v0.0.11 // indirect
	github.com/supabase-community/storage-go v0.7.0 // indirect
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80 // indirect
	go.uber.org/multierr v1.10.0 // indirect
)
