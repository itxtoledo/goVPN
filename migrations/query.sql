-- Create the rooms table
CREATE TABLE IF NOT EXISTS rooms (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  password VARCHAR(10) NOT NULL,
  public_key TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  last_active TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on last_active to speed up stale room cleanup queries
CREATE INDEX IF NOT EXISTS idx_rooms_last_active ON rooms(last_active);

-- Create unique index on public_key to ensure each public key has only one room
CREATE UNIQUE INDEX IF NOT EXISTS idx_rooms_public_key ON rooms(public_key);

-- Comment for rooms table
COMMENT ON TABLE rooms IS 'Stores information about VPN rooms created by users';
COMMENT ON COLUMN rooms.id IS 'Unique identifier for the room';
COMMENT ON COLUMN rooms.name IS 'User-friendly name of the room';
COMMENT ON COLUMN rooms.password IS 'Pin code password to access the room';
COMMENT ON COLUMN rooms.public_key IS 'Base64-encoded public key of the room creator';
COMMENT ON COLUMN rooms.created_at IS 'Timestamp when the room was created';
COMMENT ON COLUMN rooms.last_active IS 'Timestamp of the last activity in the room';

-- Create the user_rooms table to track relationships between users and rooms
CREATE TABLE IF NOT EXISTS user_rooms (
  id SERIAL PRIMARY KEY,
  room_id VARCHAR(64) NOT NULL,
  public_key TEXT NOT NULL,
  username VARCHAR(255),
  joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  last_connected TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  is_connected BOOLEAN DEFAULT TRUE,
  UNIQUE(room_id, public_key),
  FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

-- Create index on public_key for user_rooms to improve lookup performance
CREATE INDEX IF NOT EXISTS idx_user_rooms_public_key ON user_rooms(public_key);

-- Create index on room_id for user_rooms to improve lookup performance
CREATE INDEX IF NOT EXISTS idx_user_rooms_room_id ON user_rooms(room_id);

-- Comment for user_rooms table
COMMENT ON TABLE user_rooms IS 'Tracks relationships between users and the rooms they are members of';
COMMENT ON COLUMN user_rooms.id IS 'Unique identifier for the membership record';
COMMENT ON COLUMN user_rooms.room_id IS 'The ID of the room';
COMMENT ON COLUMN user_rooms.public_key IS 'Base64-encoded public key of the user';
COMMENT ON COLUMN user_rooms.username IS 'Username of the user in this room';
COMMENT ON COLUMN user_rooms.joined_at IS 'Timestamp when the user first joined the room';
COMMENT ON COLUMN user_rooms.last_connected IS 'Timestamp of the last connection to the room';
COMMENT ON COLUMN user_rooms.is_connected IS 'Whether the user is currently connected to the room';