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
5. [User Management](#user-management)
   - [Kicking a User](#kicking-a-user)
6. [WebRTC Signaling](#webrtc-signaling)
   - [Sending Offers](#sending-offers)
   - [Sending Answers](#sending-answers)
   - [Exchanging ICE Candidates](#exchanging-ice-candidates)
7. [Error Handling](#error-handling)
8. [Message ID Tracking](#message-id-tracking)
9. [Rate Limiting](#rate-limiting)
10. [Room Expiration](#room-expiration)

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

The GoVPN system uses two distinct message types:

1. **ClientMessage** - Messages sent from clients to the server (requires authentication)
2. **ServerMessage** - Messages sent from server to clients (no authentication needed)

### ClientMessage Format

All messages sent from the client to the server must include a `signature` and `publicKey` field to verify the authenticity of the sender. A `ClientMessage` has the following structure:

```json
{
  "public_key": "<base64-encoded-ed25519-public-key>",
  "signature": "<base64-encoded-signature>",
  "type": "<message-type>",
  "other_fields": "...",
}
```

### ServerMessage Format

Messages sent from the server to clients do not include authentication fields. A `ServerMessage` has the following structure:

```json
{
  "type": "<message-type>",
  "other_fields": "...",
}
```

### Client to Server Message Types

- `CreateRoom`: Create a new VPN room
- `JoinRoom`: Join an existing room
- `LeaveRoom`: Leave a room
- `Offer`: Send a WebRTC offer to a peer
- `Answer`: Send a WebRTC answer to a peer
- `Candidate`: Send an ICE candidate to a peer
- `Kick`: Kick a user from a room (room owner only)
- `Rename`: Rename a room (room owner only)

### Server to Client Message Types

- `Error`: An error occurred
- `RoomCreated`: A room was successfully created
- `RoomJoined`: Successfully joined a room
- `RoomDeleted`: A room was deleted
- `RoomRenamed`: A room was renamed
- `PeerJoined`: A new peer joined the room
- `PeerLeft`: A peer left the room
- `Kicked`: You were kicked from a room
- `KickSuccess`: Successfully kicked a user
- `RenameSuccess`: Successfully renamed a room
- `DeleteSuccess`: Successfully deleted a room
- `LeaveRoomSuccess`: Successfully left a room

## Authentication and Security

The server uses Ed25519 key pairs for room ownership verification and client authentication. All client messages must be signed with the client's private key. The client that creates a room must provide its public key during room creation. Any operations that require room ownership (such as kicking users, renaming, or deleting the room) must be signed with the corresponding private key.

### Password Requirements

Room passwords must match the following pattern: exactly 4 numeric digits (e.g., "1234").

## Room Operations

### Creating a Room

**Request (ClientMessage):**

```json
{
  "type": "CreateRoom",
  "room_name": "My VPN Room",
  "password": "1234",
  "public_key": "<base64-encoded-public-key>",
  "signature": "<base64-encoded-signature>",
  "message_id": "<unique-message-id>"
}
```

- `room_name`: A name for the room
- `password`: A password for the room (must be 4 digits)
- `public_key`: Base64-encoded Ed25519 public key
- `signature`: Base64-encoded Ed25519 signature
- `message_id`: Optional unique identifier for tracking the response

**Response (Success - ServerMessage):**

```json
{
  "type": "RoomCreated",
  "room_id": "abc123",
  "room_name": "My VPN Room",
  "password": "1234",
  "message_id": "<same-message-id-from-request>"
}
```

**Response (Error - ServerMessage):**

```json
{
  "type": "Error",
  "data": "Error message here",
  "message_id": "<same-message-id-from-request>"
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
  "type": "JoinRoom",
  "room_id": "abc123",
  "password": "1234",
  "public_key": "<base64-encoded-public-key>",
  "username": "User1",
  "signature": "<base64-encoded-signature>",
  "message_id": "<unique-message-id>"
}
```

- `room_id`: ID of the room to join
- `password`: Password for the room
- `public_key`: Base64-encoded Ed25519 public key
- `signature`: Base64-encoded Ed25519 signature
- `username`: Optional username to display
- `message_id`: Optional unique identifier for tracking the response

**Response (Success - ServerMessage):**

```json
{
  "type": "RoomJoined",
  "room_id": "abc123",
  "room_name": "My VPN Room",
  "message_id": "<same-message-id-from-request>"
}
```

**Additional Messages (to all users in the room - ServerMessage):**

```json
{
  "type": "PeerJoined",
  "room_id": "abc123",
  "public_key": "<new-user-public-key>",
  "username": "User1"
}
```

**Response (Error - ServerMessage):**

```json
{
  "type": "Error",
  "data": "Error message here",
  "message_id": "<same-message-id-from-request>"
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
  "type": "LeaveRoom",
  "room_id": "abc123",
  "public_key": "<base64-encoded-public-key>",
  "signature": "<base64-encoded-signature>",
  "message_id": "<unique-message-id>"
}
```

- `room_id`: ID of the room to leave (if not provided, the server will use the room ID associated with the connection)
- `public_key`: Base64-encoded Ed25519 public key
- `signature`: Base64-encoded Ed25519 signature
- `message_id`: Optional unique identifier for tracking the response

**Response (Success - ServerMessage):**

```json
{
  "type": "LeaveRoomSuccess",
  "room_id": "abc123",
  "message_id": "<same-message-id-from-request>"
}
```

**Note:** If the room owner leaves, the room will be deleted and all other users will be kicked with a notification.

**Message to Other Users (when owner leaves - ServerMessage):**

```json
{
  "type": "Kicked",
  "room_id": "abc123",
  "data": "Room owner has left and closed the room"
}
```

### Renaming a Room

**Request (ClientMessage):**

```json
{
  "type": "Rename",
  "room_id": "abc123",
  "room_name": "New Room Name",
  "public_key": "<base64-encoded-public-key>",
  "signature": "<base64-encoded-signature>",
  "message_id": "<unique-message-id>"
}
```

- `room_id`: ID of the room to rename
- `room_name`: New name for the room
- `public_key`: Base64-encoded Ed25519 public key
- `signature`: Base64-encoded Ed25519 signature
- `message_id`: Optional unique identifier for tracking the response

**Response (Success - ServerMessage):**

```json
{
  "type": "RenameSuccess",
  "room_id": "abc123",
  "room_name": "New Room Name",
  "message_id": "<same-message-id-from-request>"
}
```

**Additional Messages (to all users in the room - ServerMessage):**

```json
{
  "type": "RoomRenamed",
  "room_id": "abc123",
  "room_name": "New Room Name"
}
```

### Deleting a Room

Room deletion happens automatically when the owner leaves a room. There's no explicit delete message type needed.

## User Management

### Kicking a User

**Request (ClientMessage):**

```json
{
  "type": "Kick",
  "room_id": "abc123",
  "target_id": "<target-client-connection-id>",
  "public_key": "<base64-encoded-public-key>",
  "signature": "<base64-encoded-signature>",
  "message_id": "<unique-message-id>"
}
```

- `room_id`: ID of the room
- `target_id`: Connection ID of the user to kick
- `public_key`: Base64-encoded Ed25519 public key
- `signature`: Base64-encoded Ed25519 signature
- `message_id`: Optional unique identifier for tracking the response

**Response (Success - ServerMessage):**

```json
{
  "type": "KickSuccess",
  "room_id": "abc123",
  "target_id": "<target-client-connection-id>",
  "message_id": "<same-message-id-from-request>"
}
```

**Message to Kicked User (ServerMessage):**

```json
{
  "type": "Kicked",
  "room_id": "abc123"
}
```

## WebRTC Signaling

### Sending Offers

**Request (ClientMessage):**

```json
{
  "type": "Offer",
  "room_id": "abc123",
  "public_key": "<base64-encoded-public-key>",
  "destination_id": "<destination-connection-id>",
  "offer": "<webrtc-offer-string>",
  "signature": "<base64-encoded-signature>"
}
```

- `room_id`: ID of the room
- `public_key`: Sender's public key
- `destination_id`: Connection ID of the destination user
- `offer`: WebRTC offer in serialized format
- `signature`: Base64-encoded Ed25519 signature

### Sending Answers

**Request (ClientMessage):**

```json
{
  "type": "Answer",
  "room_id": "abc123",
  "public_key": "<base64-encoded-public-key>",
  "destination_id": "<destination-connection-id>",
  "answer": "<webrtc-answer-string>",
  "signature": "<base64-encoded-signature>"
}
```

- `room_id`: ID of the room
- `public_key`: Sender's public key
- `destination_id`: Connection ID of the destination user
- `answer`: WebRTC answer in serialized format
- `signature`: Base64-encoded Ed25519 signature

### Exchanging ICE Candidates

**Request (ClientMessage):**

```json
{
  "type": "Candidate",
  "room_id": "abc123",
  "public_key": "<base64-encoded-public-key>",
  "destination_id": "<destination-connection-id>",
  "candidate": "<webrtc-ice-candidate-string>",
  "signature": "<base64-encoded-signature>"
}
```

- `room_id`: ID of the room
- `public_key`: Sender's public key
- `destination_id`: Connection ID of the destination user
- `candidate`: WebRTC ICE candidate in serialized format
- `signature`: Base64-encoded Ed25519 signature

## Error Handling

Error messages have the following format (ServerMessage):

```json
{
  "type": "Error",
  "data": "Error message here",
  "message_id": "<message-id-from-original-request>"
}
```

Common error conditions include:
- Invalid room ID
- Invalid password
- Room is full
- Rate limit exceeded
- Invalid signature
- Invalid public key format
- Missing required fields

## Message ID Tracking

To track message responses, include a unique `message_id` field in your requests. The server will include the same `message_id` in the corresponding response.

```json
{
  "type": "YourMessageType",
  "message_id": "unique-id-123",
  // other fields...
}
```

## Rate Limiting

The server implements rate limiting to prevent abuse. The default rate limiting is 3 requests per minute for room creation and joining operations. Clients that exceed the rate limit will receive an `Error` message indicating that the rate limit has been exceeded.

## Room Expiration

Rooms will automatically expire after a period of inactivity (default: 30 days). The server periodically cleans up inactive rooms. Room activity is updated whenever a client joins or performs actions in the room.

## Signature Generation

To generate signatures for ClientMessages:

1. Create a copy of the message object
2. Set the `signature` field to `""` (empty string)
3. Serialize the message to JSON
4. Sign the JSON data using your Ed25519 private key
5. Base64-encode the signature
6. Add the signature to the original message

## Implementation Example (Pseudocode)

```
// Generate key pair (Ed25519)
publicKey, privateKey = generateEd25519KeyPair()
publicKeyBase64 = base64Encode(publicKey)

// Connect to the WebSocket server
websocket = new WebSocket("ws://server:port/ws")

// Create a room
clientMessage = {
  type: "CreateRoom",
  room_name: "My Room",
  password: "1234",
  public_key: publicKeyBase64,
  message_id: generateUUID()
}

// Sign the message
messageCopy = copy(clientMessage)
messageCopy.signature = ""
jsonData = JSON.stringify(messageCopy)
signature = ed25519Sign(privateKey, jsonData)
clientMessage.signature = base64Encode(signature)

websocket.send(JSON.stringify(clientMessage))

// Handle incoming messages
websocket.onMessage = function(event) {
  const serverMessage = JSON.parse(event.data)
  switch(serverMessage.type) {
    case "RoomCreated":
      console.log("Room created:", serverMessage.room_id)
      break
    case "PeerJoined":
      console.log("Peer joined:", serverMessage.public_key)
      initiateWebRTCConnection(serverMessage.public_key)
      break
    case "Offer":
      handleWebRTCOffer(serverMessage.offer, serverMessage.public_key)
      break
    // Handle other message types...
  }
}
```

## Security Considerations

1. Always validate message payloads
2. Implement proper error handling
3. Store private keys securely
4. Use HTTPS/WSS in production
5. Implement client-side rate limiting to avoid server-side rate limit errors
6. Always sign your client messages with your Ed25519 private key
7. Verify that the signature of incoming messages matches the sender's public key

---

For any questions or issues, please contact the GoVPN development team.