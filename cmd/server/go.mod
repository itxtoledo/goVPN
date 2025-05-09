module github.com/itxtoledo/govpn/cmd/server

go 1.23.0

toolchain go1.23.8

require (
	github.com/gorilla/websocket v1.5.3
	github.com/itxtoledo/govpn/libs/crypto_utils v0.0.0
	github.com/itxtoledo/govpn/libs/models v0.0.0
	github.com/joho/godotenv v1.5.1
	github.com/nedpals/supabase-go v0.5.0
	github.com/stretchr/testify v1.8.4
	github.com/supabase-community/supabase-go v0.0.4
	golang.org/x/time v0.11.0
)

replace (
	github.com/itxtoledo/govpn/libs/crypto_utils v0.0.0 => ../../libs/crypto_utils
	github.com/itxtoledo/govpn/libs/models v0.0.0 => ../../libs/models
	github.com/itxtoledo/govpn/libs/network v0.0.0 => ../../libs/network
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/supabase-community/functions-go v0.0.0-20220927045802-22373e6cb51d // indirect
	github.com/supabase-community/gotrue-go v1.2.0 // indirect
	github.com/supabase-community/postgrest-go v0.0.11 // indirect
	github.com/supabase-community/storage-go v0.7.0 // indirect
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
