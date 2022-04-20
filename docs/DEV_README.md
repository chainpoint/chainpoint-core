# Go ABCI Service

## Introduction

The Chainpoint Network's blockchain is based around the Tendermint blockchain library, a Byzantine Fault Tolerant (BFT) peer-to-peer consensus layer. Tendermint allows anyone to interact with a self-generated Tendermint blockchain using an Application-BlockChain Interface (ABCI) app written in Go.

The binary encompasses a Tendermint Core, Chainpoint ABCI app, LND, and various submodules for connecting to C_Level_DB, all running in separate threads.

This guide encompasses contributor guidelines and a brief technical overview. 

## Contributor Guidelines

We encourage anyone interested in contributing to fork this repo and submit a pull-request with desired changes. 

Please target go v1.16.8 and run gofmt on the source tree to check/fix any style issues.

All contributions and usage must adhere to the GNU aGPL v3 license.  

## Operation Modes

When initialized, Tendermint will create Genesis, Config, and Node_Key files in `~/.chainpoint/core/config`. All other data, including the tendermint db and ECDSA key files, can be found in `~/.chainpoint/core/data`.

A Tendermint full-node can become a validator, and share greater responsibilities such as confirming new Cores, generating proofs, and injecting DRAND Beacon entropy into the blockchain.

## Deeper Dive

When a Chainpoint Core starts up, it instantiates an ABCI application, Tendermint Core, and an LND node. These become bound together for the duration of operation.

Every block epoch (60 seconds by default), the ABCI application is set to perform a number of functions:

- Authenticate transactions from all other Cores.
- Aggregate all hashes received from the API into a CAL transaction and submit it to the Calendar.
- Elect a leader to send a DRAND Beacon Entropy transaction to the blockchain.
- Monitor for public keys from new Cores, which are broadcast via a JWK message.
- Monitor sync status with the rest of the Network (and shutdown critical functions if not synced yet).

Every anchor epoch (60 minutes by default), the ABCI application is set to perform the following anchor/reward functions:

- Elect a leader to anchor all hashes received since last anchor epoch by broadcasting their Merkle Root to Bitcoin via the `lnd` thread. The resulting Bitcoin TX ID is placed in a BTC-A transaction and submitted to the Calendar.
- Monitor for the confirmation of a successful anchor to Bitcoin. The resulting Bitcoin header info containing the anchor is placed in a BTC-C transaction and submitted to the Calendar.

## Anchor Interface

The `anchor` package contains an interface which allows alternate blockchain anchor engine to be implemented. 
Chainpoint's bitcoin anchoring implementation is included in the `bitcoin` subpackage. 

Any new blockchain anchor must implement the methods in the `anchor_interface.go` file, as well as assign the new anchor to the AnchorEngine in the ABCI AnchorApplication.

## Useful Packages

The following packages contain `go` language utilities which may be useful in the following ways:

- `lightning` : Methods for interacting with `tierion/lnd` modified lightning nodes. Most methods can also interact with the lightninglabs lnd nodes. 
- `aggregator` : Multithreaded method of creating Merkle trees from large numbers of hashes
- `beacon` : Retrieves timestamped entropy from the [drand](https://drand.love/) network
- `fee` : Retrieves bitcoin fees from the [bitcoinerlive](https://bitcoiner.live/) service
- `leaderelection` : Methods for deterministically electing a leader from a group of Tendermint nodes
- `merkletools` : Chainpoint Merkle tree implementation
- `tendermint_rpc` : RPC methods for interacting with a Tendermint node
- `util` : Various utilities for doing everything from verifying ECDSA signatures to generically finding array elements

## Troubleshooting

- If the Tendermint Core crashes, it will log a `panic` message. Usually this is due to the Tendermint Core having a corrupt copy of the chain. When in doubt, don't be afraid to `make clean-tendermint`, then restart. A fast-sync with the rest of the Network should fix the issue.
- If the Tendermint Core returns connection errors to other services, make sure all URIs and API keys are correct in your configuration.
- If you need to move hosts, be sure to save the `~/.chainpoint` directory and move it to the new host. You will want to adjust the Core's public IP in `~/.chainpoint/core/.env` to reflect the new host.