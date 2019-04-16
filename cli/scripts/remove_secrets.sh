#!/bin/bash
docker secret rm ETH_ADDRESS
docker secret rm ETH_PRIVATE_KEY
docker secret rm BITCOIN_WIF
docker secret rm ETH_INFURA_API_KEY
docker secret rm ETH_ETHERSCAN_API_KEY
docker secret rm ECDSA_KEYPAIR
echo -e "Wallet secret removed!"
