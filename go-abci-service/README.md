# Go ABCI Service

## Introduction

The Chainpoint Network's blockchain is based around the Tendermint blockchain library, a Byzantine Fault Tolerant (BFT) peer-to-peer consensus layer. Tendermint allows anyone to interact with a self-generated Tendermint blockchain using an Application-BlockChain Interface (ABCI) app written in Go.

This service module encompasses a Tendermint Core, Chainpoint ABCI app, and various submodules for connecting to C_Level_DB, RabbitMQ, PostgreSQL, Ethereum Smart Contracts, and Redis. The result is one dockerized binary run in a Docker Swarm capable of coordinating all Core services on a single host.

## Operation Modes

When initialized, Tendermint will create Genesis, Config, and Node_Key files in `~/.chainpoint/core/config/node_1`. When Chainpoint-Core is set to `PRIVATE_NETWORK=true` and run immediately after initialization, it will run standalone, without peering with a wider Chainpoint Network. However, when Genesis and Config files are synced to an existing Network via `make init-chain`, Chainpoint-Core will fast-sync with the Network and become a full-node capable of sharing Bitcoin anchoring responsibilities.

A Tendermint full-node can become a validator, and share greater responsibilities such as receiving hashes from Nodes, generating proofs, Minting Rewards for well-behaved Cores and Nodes, and injecting NIST Beacon entropy into the blockchain.

## Deeper Dive

When a Chainpoint Core starts up, it first retrieves all configuration options from the environment variables listed in the `swarm-compose.yaml` file in the project root. It then instantiates both an ABCI application and a Tendermint Core. These become bound together for the duration of operation.

Every block epoch (60 seconds by default), the ABCI application is set to perform a number of functions:

- Authenticate transactions from all other Cores.
- Aggregate all hashes received from `node-api-service` into a CAL transaction and submit it to the Calendar.
- Elect a leader to send a NIST Beacon Entropy transaction to the blockchain.
- Monitor for public keys from new Cores, which are broadcast via a JWK message.
- Monitor sync status with the rest of the Network (and shutdown critical functions if not synced yet).
- Receive information about new Nodes and Cores from the Chainpoint Registry ethereum smart contract.

Every anchor epoch (60 minutes by default), the ABCI application is set to perform the following anchor/reward functions:

- Elect a leader to anchor all hashes received since last anchor epoch by broadcasting their Merkle Root to Bitcoin via `btc-tx-service`. The resulting Bitcoin TX ID is placed in a BTC-A transaction and submitted to the Calendar.
- Monitor for the confirmation of a successful anchor to Bitcoin via the `btc-mon-service`. The resulting Bitcoin header info containing the anchor is placed in a BTC-C transaction and submitted to the Calendar.
- Elect a leader to audit Nodes, then reward good behavior every 24 hours. Upon successful reward, a NODE-MINT transaction is broadcast to the Calendar.

At any time, the ABCI application may:

- Sync proof data via a BTC-M message to all other Cores using Tendermint's P2P layer, so the network understands which CAL transactions are included in the anchor.
- Sync authorization data via a TOKEN message to all other Cores consisting of a hash of a Chainpoint Node's JWT auth token, so that only authorized Nodes may submit hashes to the Network.

## Troubleshooting

- If the Tendermint Core crashes, it will log a `panic` message. Usually this is due to the Tendermint Core having a corrupt copy of the chain. When in doubt, don't be afraid to `make remove` and `make clean` to stop the node and delete the chainstate, then redeploy with `make deploy`. A fast-sync with the rest of the Network should fix the issue.
- If the Tendermint Core returns connection errors to other services, make sure all URIs and API keys are correct in your configuration.
- If you need to move hosts, be sure to save the `~/.chainpoint` directory and move it to the new host. You will want to adjust the Core's public IP in `~/.chainpoint/core/.env` to reflect the new host.
