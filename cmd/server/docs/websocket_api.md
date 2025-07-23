# GoVPN WebSocket Server API Documentation

This document describes the WebSocket API for the GoVPN server, allowing developers to implement compatible clients in any programming language. The server provides a signaling mechanism for establishing computer-to-computer VPN connections between clients.

## Table of Contents

1. [Connection Establishment](#connection-establishment)
2. [Message Format](#message-format)
3. [Authentication and Security](#authentication-and-security)
4. [Network Operations](#network-operations)
   - [Creating a Network](#creating-a-network)
   - [Joining a Network](#joining-a-network)
   - [Leaving a Network](#leaving-a-network)
   - [Renaming a Network](#renaming-a-network)
   - [Deleting a Network](#deleting-a-network)
   - [Connecting to a Previously Joined Network](#connecting-to-a-previously-joined-network)
   - [Disconnecting from a Network](#disconnecting-from-a-network)
   - [Updating Client Information](#updating-client-information)
5. [Computer Management](#computer-management)
   - [Kicking a Computer](#kicking-a-computer)
6. [Connection Management](#connection-management)
   - [Ping/Pong](#pingpong)
7. [WebRTC Signaling](#webrtc-signaling)
   - [Sending Offers](#sending-offers)
   - [Sending Answers](#sending-answers)
   - [Exchanging ICE Candidates](#exchanging-ice-candidates)
8. [Error Handling](#error-handling)
9. [Message ID Tracking](#message-id-tracking)
10. [Rate Limiting](#rate-limiting)
11. [Network Expiration](#network-expiration)

## Connection Establishment

Connect to the WebSocket server using the following URL:

```
ws://<server-host>:<port>/ws
```

For secure connections:

```
wss://<server-host>:<port>/ws
```

## Message Format

The GoVPN system uses a message format that encapsulates all communications:

### SignalingMessage Format

```json
{
  "message_id": "<unique-message-id>", 
  "type": "<message-type>",
  "payload": <json-payload-as-bytes>
}
```

- `message_id`: A unique identifier for tracking request/response pairs
- `type`: The type of message (see types below)
- `payload`: The message content as a JSON object serialized to bytes

### Client to Server Message Types

- `CreateNetwork`: Create a new VPN network
- `JoinNetwork`: Join an existing network
- `ConnectNetwork`: Connect to a previously joined network without providing password again
- `DisconnectNetwork`: Temporarily disconnect from a network without leaving it
- `LeaveNetwork`: Leave a network
- `Ping`: Check connection and measure latency
- `Offer`: Send a WebRTC offer to a computer
- `Answer`: Send a WebRTC answer to a computer
- `Candidate`: Send an ICE candidate to a computer
- `Kick`: Kick a computer from a network (network owner only)
- `Rename`: Rename a network (network owner only)
- `UpdateClientInfo`: Update the client's name on the server

### Server to Client Message Types

- `Error`: An error occurred
- `NetworkCreated`: A network was successfully created
- `NetworkJoined`: Successfully joined a network
- `NetworkConnected`: Successfully connected to a previously joined network  
- `NetworkDisconnected`: Successfully disconnected from a network (but still a member)
- `NetworkDeleted`: A network was deleted
- `NetworkRenamed`: A network was renamed
- `ComputerJoined`: A new computer joined the network
- `ComputerLeft`: A computer left the network
- `ComputerConnected`: A computer connected to the network (after previously joining)
- `ComputerDisconnected`: A computer disconnected from the network (without leaving)
- `ComputerRenamed`: A computer in the network has been renamed
- `Kicked`: You were kicked from a network
- `KickSuccess`: Successfully kicked a computer
- `RenameSuccess`: Successfully renamed a network
- `DeleteSuccess`: Successfully deleted a network
- `LeaveNetworkSuccess`: Successfully left a network

## Authentication and Security

The server uses Ed25519 key pairs for network ownership verification and client authentication. All messages that require authentication must include the client's public key in the payload. The client that creates a network must provide its public key during network creation. Any operations that require network ownership (such as kicking computers, renaming, or deleting the network) must be performed using the same public key used to create the network.

### Password Requirements

Network passwords must match the following pattern: exactly 4 numeric digits (e.g., "1234").

## Network Operations

### Creating a Network

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "CreateNetwork",
  "payload": {
    "network_name": "My VPN Network",
    "password": "1234",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

- `network_name`: A name for the network
- `password`: A password for the network (must be 4 digits)
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "NetworkCreated",
  "payload": {
    "network_id": "abc123",
    "network_name": "My VPN Network",
    "password": "1234",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

**Response (Error - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "Error",
  "payload": {
    "error": "Error message here"
  }
}
```

Common error messages include:
- "Network name, password, and public key are required"
- "Password does not match required pattern" 
- "This public key has already created network: {networkID}"
- "Network ID conflict, please try again"
- "Invalid public key format"
- "Error creating network in database"

### Joining a Network

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "JoinNetwork",
  "payload": {
    "network_id": "abc123",
    "password": "1234",
    "public_key": "<base64-encoded-public-key>",
    "computername": "Computer1"
  }
}
```

- `network_id`: ID of the network to join
- `password`: Password for the network
- `public_key`: Base64-encoded Ed25519 public key
- `computername`: Optional computername to display

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "NetworkJoined",
  "payload": {
    "network_id": "abc123",
    "network_name": "My VPN Network"
  }
}
```

**Additional Messages (to all computers in the network - ServerMessage):**

```json
{
  "type": "ComputerJoined",
  "payload": {
    "network_id": "abc123",
    "public_key": "<new-computer-public-key>",
    "computername": "Computer1"
  }
}
```

**Response (Error - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "Error",
  "payload": {
    "error": "Error message here"
  }
}
```

Common error messages include:
- "Network does not exist"
- "Incorrect password"
- "Public key is required"
- "Network is full"
- "Rate limit exceeded. Please try again later."

### Leaving a Network

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "LeaveNetwork",
  "payload": {
    "network_id": "abc123",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

- `network_id`: ID of the network to leave (if not provided, the server will use the network ID associated with the connection)
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "LeaveNetwork",
  "payload": {
    "network_id": "abc123"
  }
}
```

**Note:** If the network owner leaves, the network will be deleted and all other computers will be kicked with a notification.

**Message to Other Computers (when owner leaves - ServerMessage):**

```json
{
  "type": "NetworkDeleted",
  "payload": {
    "network_id": "abc123"
  }
}
```

### Renaming a Network

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Rename",
  "payload": {
    "network_id": "abc123",
    "network_name": "New Network Name",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

- `network_id`: ID of the network to rename
- `network_name`: New name for the network
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "RenameSuccess",
  "payload": {
    "network_id": "abc123",
    "network_name": "New Network Name"
  }
}
```

**Additional Messages (to all computers in the network - ServerMessage):**

```json
{
  "type": "NetworkRenamed",
  "payload": {
    "network_id": "abc123",
    "network_name": "New Network Name"
  }
}
```

### Deleting a Network

Network deletion happens automatically when the owner leaves a network. There's no explicit delete message type needed.

### Connecting to a Previously Joined Network

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "ConnectNetwork",
  "payload": {
    "network_id": "abc123",
    "public_key": "<base64-encoded-public-key>",
    "computername": "Computer1"
  }
}
```

- `network_id`: ID of the network to connect to
- `public_key`: Base64-encoded Ed25519 public key
- `computername`: Optional computername to display

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "NetworkConnected",
  "payload": {
    "network_id": "abc123",
    "network_name": "My VPN Network"
  }
}
```

**Additional Messages (to all computers in the network - ServerMessage):**

```json
{
  "type": "ComputerConnected",
  "payload": {
    "network_id": "abc123",
    "public_key": "<computer-public-key>",
    "computername": "Computer1"
  }
}
```

**Response (Error - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "Error",
  "payload": {
    "error": "Error message here"
  }
}
```

Common error messages include:
- "Network does not exist"
- "Public key is required"
- "Network is full"

### Disconnecting from a Network (without leaving it)

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "DisconnectNetwork",
  "payload": {
    "network_id": "abc123",
    "public_key": "<base64-encoded-public_key>"
  }
}
```

- `network_id`: ID of the network to disconnect from (if not provided, the server will use the network ID associated with the connection)
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "NetworkDisconnected",
  "payload": {
    "network_id": "abc123"
  }
}
```

**Additional Messages (to all computers in the network - ServerMessage):**

```json
{
  "type": "ComputerDisconnected",
  "payload": {
    "network_id": "abc123",
    "public_key": "<disconnected-computer-public-key>"
  }
}
```

### Updating Client Information

This message is sent by the client to update its `computername` across all networks it has joined. The server will update the database and then send an updated list of networks to the client.

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "UpdateClientInfo",
  "payload": {
    "public_key": "<base64-encoded-public-key>",
    "client_name": "NewClientName"
  }
}
```

- `public_key`: Base64-encoded Ed25519 public key of the client.
- `client_name`: The new name for the client.

**Response (Success - ServerMessage):**

Upon successful update, the server will send a `ComputerNetworks` message back to the client with the updated information.

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "ComputerNetworks",
  "payload": {
    "networks": [
      // ... updated list of networks with the new client name
    ]
  }
}
```

**Response (Error - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "Error",
  "payload": {
    "error": "Error message here"
  }
}
```

Common error messages include:
- "Public key is required for updating client info"
- "Client name is required"
- "Error updating client name"

## Computer Management

### Kicking a Computer

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Kick",
  "payload": {
    "network_id": "abc123",
    "target_id": "<target-client-connection-id>",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

- `network_id`: ID of the network
- `target_id`: Connection ID of the computer to kick
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "KickSuccess",
  "payload": {
    "network_id": "abc123",
    "target_id": "<target-client-connection-id>"
  }
}
```

**Message to Kicked Computer (ServerMessage):**

```json
{
  "type": "Kicked",
  "payload": {
    "network_id": "abc123"
  }
}
```

## Connection Management

### Ping/Pong

The ping/pong mechanism allows clients to verify their connection to the server and measure latency.

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Ping",
  "payload": {
    "timestamp": <current-unix-timestamp>,
    "public_key": "<base64-encoded-public-key>",
    "action": "ping"
  }
}
```

- `timestamp`: Current client timestamp (in nanoseconds, Unix format)
- `public_key`: Base64-encoded Ed25519 public key (optional)
- `action`: Set to "ping" to identify the request type

**Response (ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "Ping",
  "payload": {
    "client_timestamp": <original-timestamp-from-request>,
    "server_timestamp": <current-server-timestamp>,
    "status": "ok"
  }
}
```

- `client_timestamp`: The original timestamp sent by the client
- `server_timestamp`: Current server timestamp (in nanoseconds, Unix format)
- `status`: Always "ok" if the ping was successful

## WebRTC Signaling

### Sending Offers

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Offer",
  "payload": {
    "network_id": "abc123",
    "public_key": "<base64-encoded-public-key>",
    "destination_id": "<destination-connection-id>",
    "offer": "<webrtc-offer-string>"
  }
}
```

- `network_id`: ID of the network
- `public_key`: Sender's public key
- `destination_id`: Connection ID of the destination computer
- `offer`: WebRTC offer in serialized format

### Sending Answers

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Answer",
  "payload": {
    "network_id": "abc123",
    "public_key": "<base64-encoded-public-key>",
    "destination_id": "<destination-connection-id>",
    "answer": "<webrtc-answer-string>"
  }
}
```

- `network_id`: ID of the network
- `public_key`: Sender's public key
- `destination_id`: Connection ID of the destination computer
- `answer`: WebRTC answer in serialized format

### Exchanging ICE Candidates

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Candidate",
  "payload": {
    "network_id": "abc123",
    "public_key": "<base64-encoded-public-key>",
    "destination_id": "<destination-connection-id>",
    "candidate": "<webrtc-ice-candidate-string>"
  }
}
```

- `network_id`: ID of the network
- `public_key`: Sender's public key
- `destination_id`: Connection ID of the destination computer
- `candidate`: WebRTC ICE candidate in serialized format

## Error Handling

Error messages have the following format (ServerMessage):

```json
{
  "message_id": "<message-id-from-original-request>",
  "type": "Error",
  "payload": {
    "error": "Error message here"
  }
}
```

Common error conditions include:
- Invalid network ID
- Invalid password
- Network is full
- Rate limit exceeded
- Invalid public key format
- Missing required fields

## Message ID Tracking

To track message responses, include a unique `message_id` field in your requests. The server will include the same `message_id` in the corresponding response.

```json
{
  "message_id": "unique-id-123",
  "type": "YourMessageType",
  "payload": {
    // other fields...
  }
}
```

## Rate Limiting

The server implements rate limiting to prevent abuse. The default rate limiting is 3 requests per minute for network creation and joining operations. Clients that exceed the rate limit will receive an `Error` message indicating that the rate limit has been exceeded.

## Network Expiration

Networks will automatically expire after a period of inactivity (default: 30 days). The server periodically cleans up inactive networks. Network activity is updated whenever a client joins or performs actions in the network.

## Implementation Example (Pseudocode)

```
// Generate key pair (Ed25519)
publicKey, privateKey = generateEd25519KeyPair()
publicKeyBase64 = base64Encode(publicKey)

// Connect to the WebSocket server
websocket = new WebSocket("ws://server:port/ws")

// Create a network
messageID = generateUUID()
payload = {
  network_name: "My Network",
  password: "1234",
  public_key: publicKeyBase64
}

message = {
  message_id: messageID,
  type: "CreateNetwork",
  payload: JSON.stringify(payload)
}

websocket.send(JSON.stringify(message))

// Handle incoming messages
websocket.onMessage = function(event) {
  const serverMessage = JSON.parse(event.data)
  
  switch(serverMessage.type) {
    case "NetworkCreated":
      const responsePayload = JSON.parse(serverMessage.payload)
      console.log("Network created:", responsePayload.network_id)
      break
      
    case "ComputerJoined":
      const computerPayload = JSON.parse(serverMessage.payload)
      console.log("Computer joined:", computerPayload.public_key)
      initiateWebRTCConnection(computerPayload.public_key)
      break
      
    case "Ping":
      const pingResponse = JSON.parse(serverMessage.payload)
      const latency = (Date.now() - pingResponse.client_timestamp) / 1000000 // ms
      console.log("Ping latency:", latency, "ms")
      break
      
    // Handle other message types...
  }
}

// To check connection and measure latency
function sendPing() {
  const messageID = generateUUID()
  const pingPayload = {
    timestamp: Date.now(),
    action: "ping",
    public_key: publicKeyBase64
  }
  
  const message = {
    message_id: messageID,
    type: "Ping",
    payload: JSON.stringify(pingPayload)
  }
  
  websocket.send(JSON.stringify(message))
}
```

## Security Considerations

1. Always validate message payloads
2. Implement proper error handling
3. Store private keys securely
4. Use HTTPS/WSS in production
5. Implement client-side rate limiting to avoid server-side rate limit errors
6. Regularly send ping messages to keep the connection alive
7. Handle connection errors and implement reconnection logic

---

For any questions or issues, please contact the GoVPN development team.