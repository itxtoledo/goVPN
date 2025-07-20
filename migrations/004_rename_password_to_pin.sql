-- Migration to rename the 'password' column to 'pin' in the 'networks' table

ALTER TABLE networks
RENAME COLUMN password TO pin;

-- Update the comment for the renamed column
COMMENT ON COLUMN networks.pin IS 'PIN code to access the network';
