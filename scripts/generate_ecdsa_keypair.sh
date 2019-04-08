#!/bin/bash
openssl ecparam -genkey -name secp256r1 -noout -out data/keys/ecdsa_key.pem
cat data/keys/ecdsa_key.pem | docker secret create ECDSA_KEYPAIR - || echo -e "can't create secret in swarm"
echo -e "ECDSA Keypair created!"
