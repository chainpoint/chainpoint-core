#!/bin/bash
docker secret rm ETH_PRIVATE_KEY
docker secret rm BITCOIN_WIF
docker secret rm ECDSA_PKPEM
echo -e "Wallet secret removed!"
