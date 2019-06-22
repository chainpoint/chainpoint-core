# Go ABCI Service

## Introduction

The Chainpoint Network's blockchain is based around the Tendermint blockchain library, a Byzantine Fault Tolerant (BFT) peer-to-peer consensus layer. Tendermint allows anyone to interact with a self-generated Tendermint blockchain using an Application-BlockChain Interface (ABCI) app written in Go.

This service module encompasses a Tendermint Node, Chainpoint ABCI app, and various submodules for connecting to C_Level_DB, RabbitMQ, PostgreSQL, Ethereum Smart Contracts, and Redis. The result is one binary run in a docker container capable of coordinating all Core services on a single host.

## Operation

When initialized, Tendermint will create Genesis, Config, and Node_Key files in `~/.chainpoint/core/config/node_1`. When Chainpoint-Core is run immediately after initialization, it will run standalone, without peering with a wider Chainpoint Network.  
However, when Genesis and Config files are synced to an existing Network via `make init-chain`, Chainpoint-Core will fast-sync with the Network and become a full-node capable of sharing Bitcoin anchoring responsibilities.
