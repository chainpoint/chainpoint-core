#!/bin/bash
docker secret rm HOT_WALLET_PASS
docker secret rm HOT_WALLET_ADDRESS
docker secret rm ECDSA_PKPEM
docker secret rm SESSION_SECRET
echo -e "Wallet secret removed!"
