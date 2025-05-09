# GoVPN WebSocket Server API Documentation

This document describes the WebSocket API for the GoVPN server, allowing developers to implement compatible clients in any programming language. The server provides a signaling mechanism for establishing peer-to-peer VPN connections between clients.

## Table of Contents

1. [Connection Establishment](#connection-establishment)
2. [Message Format](#message-format)
3. [Authentication and Security](#authentication-and-security)
4. [Room Operations](#room-operations)
   - [Creating a Room](#creating-a-room)
   - [Joining a Room](#joining-a-room)
   - [Leaving a Room](#leaving-a-room)
   - [Renaming a Room](#renaming-a-room)
   - [Deleting a Room](#deleting-a-room)
   - [Connecting to a Previously Joined Room](#connecting-to-a-previously-joined-room)
   - [Disconnecting from a Room](#disconnecting-from-a-room)
5. [User Management](#user-management)
   - [Kicking a User](#kicking-a-user)
6. [Connection Management](#connection-management)
   - [Ping/Pong](#pingpong)
7. [WebRTC Signaling](#webrtc-signaling)
   - [Sending Offers](#sending-offers)
   - [Sending Answers](#sending-answers)
   - [Exchanging ICE Candidates](#exchanging-ice-candidates)
8. [Error Handling](#error-handling)
9. [Message ID Tracking](#message-id-tracking)
10. [Rate Limiting](#rate-limiting)
11. [Room Expiration](#room-expiration)

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

- `CreateRoom`: Create a new VPN room
- `JoinRoom`: Join an existing room
- `ConnectRoom`: Connect to a previously joined room without providing password again
- `DisconnectRoom`: Temporarily disconnect from a room without leaving it
- `LeaveRoom`: Leave a room
- `Ping`: Check connection and measure latency
- `Offer`: Send a WebRTC offer to a peer
- `Answer`: Send a WebRTC answer to a peer
- `Candidate`: Send an ICE candidate to a peer
- `Kick`: Kick a user from a room (room owner only)
- `Rename`: Rename a room (room owner only)

### Server to Client Message Types

- `Error`: An error occurred
- `RoomCreated`: A room was successfully created
- `RoomJoined`: Successfully joined a room
- `RoomConnected`: Successfully connected to a previously joined room  
- `RoomDisconnected`: Successfully disconnected from a room (but still a member)
- `RoomDeleted`: A room was deleted
- `RoomRenamed`: A room was renamed
- `PeerJoined`: A new peer joined the room
- `PeerLeft`: A peer left the room
- `PeerConnected`: A peer connected to the room (after previously joining)
- `PeerDisconnected`: A peer disconnected from the room (without leaving)
- `Kicked`: You were kicked from a room
- `KickSuccess`: Successfully kicked a user
- `RenameSuccess`: Successfully renamed a room
- `DeleteSuccess`: Successfully deleted a room
- `LeaveRoomSuccess`: Successfully left a room

## Authentication and Security

The server uses Ed25519 key pairs for room ownership verification and client authentication. All messages that require authentication must include the client's public key in the payload. The client that creates a room must provide its public key during room creation. Any operations that require room ownership (such as kicking users, renaming, or deleting the room) must be performed using the same public key used to create the room.

### Password Requirements

Room passwords must match the following pattern: exactly 4 numeric digits (e.g., "1234").

## Room Operations

### Creating a Room

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "CreateRoom",
  "payload": {
    "room_name": "My VPN Room",
    "password": "1234",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

- `room_name`: A name for the room
- `password`: A password for the room (must be 4 digits)
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "RoomCreated",
  "payload": {
    "room_id": "abc123",
    "room_name": "My VPN Room",
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
- "Room name, password, and public key are required"
- "Password does not match required pattern" 
- "This public key has already created room: {roomID}"
- "Room ID conflict, please try again"
- "Invalid public key format"
- "Error creating room in database"

### Joining a Room

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "JoinRoom",
  "payload": {
    "room_id": "abc123",
    "password": "1234",
    "public_key": "<base64-encoded-public-key>",
    "username": "User1"
  }
}
```

- `room_id`: ID of the room to join
- `password`: Password for the room
- `public_key`: Base64-encoded Ed25519 public key
- `username`: Optional username to display

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "RoomJoined",
  "payload": {
    "room_id": "abc123",
    "room_name": "My VPN Room"
  }
}
```

**Additional Messages (to all users in the room - ServerMessage):**

```json
{
  "type": "PeerJoined",
  "payload": {
    "room_id": "abc123",
    "public_key": "<new-user-public-key>",
    "username": "User1"
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
- "Room does not exist"
- "Incorrect password"
- "Public key is required"
- "Room is full"
- "Rate limit exceeded. Please try again later."

### Leaving a Room

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "LeaveRoom",
  "payload": {
    "room_id": "abc123",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

- `room_id`: ID of the room to leave (if not provided, the server will use the room ID associated with the connection)
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "LeaveRoom",
  "payload": {
    "room_id": "abc123"
  }
}
```

**Note:** If the room owner leaves, the room will be deleted and all other users will be kicked with a notification.

**Message to Other Users (when owner leaves - ServerMessage):**

```json
{
  "type": "RoomDeleted",
  "payload": {
    "room_id": "abc123"
  }
}
```

### Renaming a Room

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Rename",
  "payload": {
    "room_id": "abc123",
    "room_name": "New Room Name",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

- `room_id`: ID of the room to rename
- `room_name`: New name for the room
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "RenameSuccess",
  "payload": {
    "room_id": "abc123",
    "room_name": "New Room Name"
  }
}
```

**Additional Messages (to all users in the room - ServerMessage):**

```json
{
  "type": "RoomRenamed",
  "payload": {
    "room_id": "abc123",
    "room_name": "New Room Name"
  }
}
```

### Deleting a Room

Room deletion happens automatically when the owner leaves a room. There's no explicit delete message type needed.

### Connecting to a Previously Joined Room

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "ConnectRoom",
  "payload": {
    "room_id": "abc123",
    "public_key": "<base64-encoded-public-key>",
    "username": "User1"
  }
}
```

- `room_id`: ID of the room to connect to
- `public_key`: Base64-encoded Ed25519 public key
- `username`: Optional username to display

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "RoomConnected",
  "payload": {
    "room_id": "abc123",
    "room_name": "My VPN Room"
  }
}
```

**Additional Messages (to all users in the room - ServerMessage):**

```json
{
  "type": "PeerConnected",
  "payload": {
    "room_id": "abc123",
    "public_key": "<user-public-key>",
    "username": "User1"
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
- "Room does not exist"
- "Public key is required"
- "Room is full"

### Disconnecting from a Room (without leaving it)

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "DisconnectRoom",
  "payload": {
    "room_id": "abc123",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

- `room_id`: ID of the room to disconnect from (if not provided, the server will use the room ID associated with the connection)
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "RoomDisconnected",
  "payload": {
    "room_id": "abc123"
  }
}
```

**Additional Messages (to all users in the room - ServerMessage):**

```json
{
  "type": "PeerDisconnected",
  "payload": {
    "room_id": "abc123",
    "public_key": "<disconnected-user-public-key>"
  }
}
```

## User Management

### Kicking a User

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Kick",
  "payload": {
    "room_id": "abc123",
    "target_id": "<target-client-connection-id>",
    "public_key": "<base64-encoded-public-key>"
  }
}
```

- `room_id`: ID of the room
- `target_id`: Connection ID of the user to kick
- `public_key`: Base64-encoded Ed25519 public key

**Response (Success - ServerMessage):**

```json
{
  "message_id": "<same-message-id-from-request>",
  "type": "KickSuccess",
  "payload": {
    "room_id": "abc123",
    "target_id": "<target-client-connection-id>"
  }
}
```

**Message to Kicked User (ServerMessage):**

```json
{
  "type": "Kicked",
  "payload": {
    "room_id": "abc123"
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
    "room_id": "abc123",
    "public_key": "<base64-encoded-public-key>",
    "destination_id": "<destination-connection-id>",
    "offer": "<webrtc-offer-string>"
  }
}
```

- `room_id`: ID of the room
- `public_key`: Sender's public key
- `destination_id`: Connection ID of the destination user
- `offer`: WebRTC offer in serialized format

### Sending Answers

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Answer",
  "payload": {
    "room_id": "abc123",
    "public_key": "<base64-encoded-public-key>",
    "destination_id": "<destination-connection-id>",
    "answer": "<webrtc-answer-string>"
  }
}
```

- `room_id`: ID of the room
- `public_key`: Sender's public key
- `destination_id`: Connection ID of the destination user
- `answer`: WebRTC answer in serialized format

### Exchanging ICE Candidates

**Request (ClientMessage):**

```json
{
  "message_id": "<unique-message-id>",
  "type": "Candidate",
  "payload": {
    "room_id": "abc123",
    "public_key": "<base64-encoded-public-key>",
    "destination_id": "<destination-connection-id>",
    "candidate": "<webrtc-ice-candidate-string>"
  }
}
```

- `room_id`: ID of the room
- `public_key`: Sender's public key
- `destination_id`: Connection ID of the destination user
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
- Invalid room ID
- Invalid password
- Room is full
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

The server implements rate limiting to prevent abuse. The default rate limiting is 3 requests per minute for room creation and joining operations. Clients that exceed the rate limit will receive an `Error` message indicating that the rate limit has been exceeded.

## Room Expiration

Rooms will automatically expire after a period of inactivity (default: 30 days). The server periodically cleans up inactive rooms. Room activity is updated whenever a client joins or performs actions in the room.

## Implementation Example (Pseudocode)

```
// Generate key pair (Ed25519)
publicKey, privateKey = generateEd25519KeyPair()
publicKeyBase64 = base64Encode(publicKey)

// Connect to the WebSocket server
websocket = new WebSocket("ws://server:port/ws")

// Create a room
messageID = generateUUID()
payload = {
  room_name: "My Room",
  password: "1234",
  public_key: publicKeyBase64
}

message = {
  message_id: messageID,
  type: "CreateRoom",
  payload: JSON.stringify(payload)
}

websocket.send(JSON.stringify(message))

// Handle incoming messages
websocket.onMessage = function(event) {
  const serverMessage = JSON.parse(event.data)
  
  switch(serverMessage.type) {
    case "RoomCreated":
      const responsePayload = JSON.parse(serverMessage.payload)
      console.log("Room created:", responsePayload.room_id)
      break
      
    case "PeerJoined":
      const peerPayload = JSON.parse(serverMessage.payload)
      console.log("Peer joined:", peerPayload.public_key)
      initiateWebRTCConnection(peerPayload.public_key)
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