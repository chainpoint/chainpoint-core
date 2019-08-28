#!/bin/bash
docker secret rm BITCOIN_WIF
docker secret rm HOT_WALLET_PASS
docker secret rm HOT_WALLET_ADDRESS
echo -e "Wallet secret removed!"
