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