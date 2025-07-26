#!/bin/bash

# Function to kill all background processes on exit
cleanup() {
    echo "Terminating clients..."
    kill $(jobs -p) 2>/dev/null
    exit 0
}

# Setup trap to catch Ctrl+C and other termination signals
trap cleanup SIGINT SIGTERM

# Start the first client
go run . --config tmp/client1 &
PID1=$!

# Start the second client
go run . --config tmp/client2 &
PID2=$!

echo "Both clients started. Press Ctrl+C to terminate both."

# Wait for all background processes to complete
# This keeps the script running and able to catch signals
wait
