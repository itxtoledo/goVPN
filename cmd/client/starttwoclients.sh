#/bin/bash

# Start the first client
go run . --config tmp/client1 &

# Start the second client
go run . --config tmp/client2 &
