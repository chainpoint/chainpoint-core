#!/bin/bash

source ./env_secrets_expand.sh
cat /run/secrets/ECDSA_KEYPAIR
./abci-service