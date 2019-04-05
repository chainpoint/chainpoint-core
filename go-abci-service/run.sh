#!/bin/bash

source ./env_secrets_expand.sh
tendermint node $PEERS --moniker=`hostname` --proxy_app=abci:26658