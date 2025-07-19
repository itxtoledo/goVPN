ALTER TABLE computer_networks
ADD COLUMN peer_ip VARCHAR(15);

ALTER TABLE computer_networks
ADD CONSTRAINT unique_network_id_peer_ip UNIQUE (network_id, peer_ip);