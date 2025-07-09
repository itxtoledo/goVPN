-- Create the networks table
CREATE TABLE IF NOT EXISTS networks (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  password VARCHAR(10) NOT NULL,
  public_key TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  last_active TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on last_active to speed up stale network cleanup queries
CREATE INDEX IF NOT EXISTS idx_networks_last_active ON networks(last_active);

-- Create unique index on public_key to ensure each public key has only one network
CREATE UNIQUE INDEX IF NOT EXISTS idx_networks_public_key ON networks(public_key);

-- Comment for networks table
COMMENT ON TABLE networks IS 'Stores information about VPN networks created by computers';
COMMENT ON COLUMN networks.id IS 'Unique identifier for the network';
COMMENT ON COLUMN networks.name IS 'Computer-friendly name of the network';
COMMENT ON COLUMN networks.password IS 'Pin code password to access the network';
COMMENT ON COLUMN networks.public_key IS 'Base64-encoded public key of the network creator';
COMMENT ON COLUMN networks.created_at IS 'Timestamp when the network was created';
COMMENT ON COLUMN networks.last_active IS 'Timestamp of the last activity in the network';

-- Create the computer_networks table to track relationships between computers and networks
CREATE TABLE IF NOT EXISTS computer_networks (
  id SERIAL PRIMARY KEY,
  network_id VARCHAR(64) NOT NULL,
  public_key TEXT NOT NULL,
  computername VARCHAR(255),
  joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  last_connected TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  is_connected BOOLEAN DEFAULT TRUE,
  UNIQUE(network_id, public_key),
  FOREIGN KEY (network_id) REFERENCES networks(id) ON DELETE CASCADE
);

-- Create index on public_key for computer_networks to improve lookup performance
CREATE INDEX IF NOT EXISTS idx_computer_networks_public_key ON computer_networks(public_key);

-- Create index on network_id for computer_networks to improve lookup performance
CREATE INDEX IF NOT EXISTS idx_computer_networks_network_id ON computer_networks(network_id);

-- Comment for computer_networks table
COMMENT ON TABLE computer_networks IS 'Tracks relationships between computers and the networks they are members of';
COMMENT ON COLUMN computer_networks.id IS 'Unique identifier for the membership record';
COMMENT ON COLUMN computer_networks.network_id IS 'The ID of the network';
COMMENT ON COLUMN computer_networks.public_key IS 'Base64-encoded public key of the computer';
COMMENT ON COLUMN computer_networks.computername IS 'ComputerName of the computer in this network';
COMMENT ON COLUMN computer_networks.joined_at IS 'Timestamp when the computer first joined the network';
COMMENT ON COLUMN computer_networks.last_connected IS 'Timestamp of the last connection to the network';
COMMENT ON COLUMN computer_networks.is_connected IS 'Whether the computer is currently connected to the network';