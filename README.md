# Chainpoint Core

[![JavaScript Style Guide](https://img.shields.io/badge/code_style-standard-brightgreen.svg)](https://standardjs.com)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

Chainpoint Core is an aggregation and anchoring service that collects hash input from Chainpoint Nodes via a web API, aggregates the hashes into a single root hash, then periodically commits the root hash to the Bitcoin blockchain as a form of distributed digital notarization.

## Important Notice

This software is intended to be run as part of the Core Network of the Chainpoint Network. It is for operators wanting to help run the anchoring service. If you are interested in running a Chainpoint Node, or installing a copy of our command line interface please instead visit:

[https://github.com/chainpoint/chainpoint-node-src](https://github.com/chainpoint/chainpoint-node-src)

[https://github.com/chainpoint/chainpoint-cli](https://github.com/chainpoint/chainpoint-cli)

## Quick Start

You can find a script that will install all prerequisite dependencies on Mac and Linux [here](https://github.com/chainpoint/chainpoint-core/blob/master/cli/scripts/install_deps.sh).

Build and start the whole system locally with `make`. Running `make help`
will display additional `Makefile` commands that are available.

```sh
git clone https://github.com/chainpoint/chainpoint-core
cd chainpoint-core
make init           #interactive
make register   #Stake with public smart contract
make deploy     #Deploy Chainpoint Core to docker swarm
```

## Introduction

Chainpoint-Core serves as an intermediate layer between hash aggregators (Chainpoint Nodes) and Bitcoin, and hence as the top-most layer of the Chainpoint Network.
Hashes submitted by Nodes are aggregated and periodically broadcast to a Tendermint-based blockchain, the Calendar, created by consensus of all Cores.
Every hour, a Core is elected to anchor the state of the Calendar to Bitcoin.

To connect to an existing Chainpoint blockchain, set the PEERS environment variable in the .env file to a comma-delimited list of `<tendermint ID>@<ip>` pairs. The ID of a given Core can be found by visiting `<ip>/status`

### Configuration

Chainpoint Core requires a minimum of 2 CPU Cores and 16 GB of RAM to run. Hard disk size should be at least 500 GB for the chain data (or be expandable).

You will need to set up a configuration and secrets (bitcoin and ethereum) before running. `make init` will do most of the heavy lifting for you.

Chainpoint Core currently uses Docker Swarm when running in Production mode. Running `make init` will initialize a Docker Swarm node on the host machine and prompt the user for secrets to be stored in Swarm's secrets system.
This command will also copy `.env.sample` to `.env`. The `.env` file will be used by `docker-compose` to set required environment variables.

There are further settings found in the `.env.sample` and `swarm-compose.yaml` file.
These are more permanent and altering them may cause problems connecting to the public Chainpoint testnets and mainnet.
However, they may be invaluable for setting up a private Chainpoint Network with different parameters, for example by configuring more frequent bitcoin anchoring or excluding the smart contract registration requirement.

The following are the descriptions of the configuration parameters:

| Name                     | Type    | Location                     | Description                                                                                                                                      |
| :----------------------- | :------ | :--------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------- |
| CHAINPOINT_CORE_BASE_URI | String  | .env                         | Public URI of host machine, of the form `http://35.245.53.181`                                                                                   |
| PRIVATE_NETWORK          | Boolean | .env                         | Sets Core to use pre-seeded list of Nodes instead of registry smart contract discovery. Default is false.                                        |
| NETWORK                  | String  | .env                         | Set to `testnet` to use Bitcoin and Ethereum testnets. Default is `mainnet`.                                                                     |
| PRIVATE_NODE_IPS         | String  | .env                         | Comma-delimited list of private Nodes for use with PRIVATE_NETWORK. Default is empty string.                                                     |
| NODE_ENV                 | String  | .env                         | Sets Core to use either ethereum/bitcoin mainnets (`production`) or testnets (`development`). Defaults to `production`                           |
| ANCHOR_INTERVAL          | String  | swarm-compose.yaml           | how often, in block time, the Core network should be anchored to Bitccoin. Default is 60.                                                        |
| HASHES_PER_MERKLE_TREE   | String  | swarm-compose.yaml           | maximum number of hashes the aggregation process will consume per aggregation interval. Default is 250000                                        |
| PEERS                    | String  | .env                         | Comma-delimited list of Tendermint peer URIs of the form $ID@$IP:\$Port, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| SEEDS                    | String  | .env                         | Comma-delimited list of Tendermint seed URIs of the form $ID@$IP:\$Port, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| ETH_INFURA_API_KEY       | String  | Docker Secrets (`make init`) | API key to use Infura ethereum web services                                                                                                      |
| ETH_ETHERSCAN_API_KEY    | String  | Docker Secrets (`make init`) | API key to use etherscan ethereum web services as a fallback to infura                                                                           |
| ETH_PRIVATE_KEY          | String  | Docker Secrets (`make init`) | Private key for this Core's Ethereum account.                                                                                                    |
| ECDSA_PKPEM              | String  | Docker Secrets (`make init`) | Keypair used to create JWKs for Core's API auth                                                                                                  |
| BITCOIN_WIF              | String  | Docker Secrets (`make init`) | Private key for bitcoin hotwallet, used to paying anchoring fees                                                                                 |
| AGGREGATE                | Boolean | swarm-compose.yaml           | Whether to aggregate hashes and send them to the Calendar blockchain. Defaults to true                                                           |
| ANCHOR                   | Boolean | swarm-compose.yaml           | Whether to anchor the state of the Calendar to Bitcoin                                                                                           |
| LOG_FILTER               | String  | swarm-compose.yaml           | Log Verbosity. Defaults to `"main:debug,state:info,*:error"`                                                                                     |
| LOG_LEVEL                | String  | swarm-compose.yaml           | Level of detail included in Logs. Defaults to `info`                                                                                             |

## Startup

To start up a Core node without connecting to the rest of the Chainpoint Network:

1. Run `make init` to initialize the configuration directory

2. Run `make register` to submit your Core as a candidate to join the Chainpoint Network

3. Run `make deploy` to download all containers and start all services.

If startup is successful, running `docker service logs -f chainpoint-core_abci` will show the log message `Executed block` every minute.

## Joining the Chainpoint Testnet

After running `make init`, you can join the public testnet as a Full Node:

1. Run `make init-chain` to download the testnet genesis and config files

2. Specify peers by adding `PEERS="73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656,cbc0f714430ccf30e2a5cfb3da60c26d317abe72@35.188.238.186:26656"` to the .env file in the root project directory

3. Run `make deploy` to copy the .env file to the config directory and deploy Core.

## Upgrade

You can upgrade Core by running `make clean-tendermint` and `docker-compose pull`, then by redeploying with `make deploy`.

## Build

### Build for local `docker-compose`

`make build`

### Build for GCR / DockerHub

Edit the `image:` keys for each service in the docker-compose file to reflect your desired docker repo. Run `make build`, authenticate with your docker host service, then run `docker-compose push`.

## Documentation

READMEs for each Core micro-service are available:

| Service                  | Description                                                                                                              | Readme                                                                                                 |
| :----------------------- | :----------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------- |
| go-abci-service          | Runs the Tendermint blockchain service and coordinates all Core activity                                                 | [README](https://github.com/chainpoint/chainpoint-core/blob/master/go-abci-service/README.md)          |
| node-api-service         | Web API for interacting with Chainpoint-Nodes                                                                            | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-api-service/README.md)         |
| node-btc-tx-service      | Transmits a Merkle Root to Bitcoin and returns the Bitcoin TX ID                                                         | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-btc-tx-service/README.md)      |
| node-btc-mon-service     | Monitors the above Bitcoin TX for 6 confirmations and informs go-abci-service when complete                              | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-btc-mon-service/README.md)     |
| node-proof-gen-service   | Generates cryptographic proofs demonstrating how Chainpoint-Node data is included in the Chainpoint Calendar and Bitcoin | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-gen-service/README.md)   |
| node-proof-state-service |                                                                                                                          | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-state-service/README.md) |

## Development

We encourage anyone interested in contributing to fork this repo and submit a pull-request with desired changes.

`make dev` will bring up a docker-compose instance geared toward development. API will be accessible on port 80, while Tendermint is accessible on ports 26656-26657.

## License

[GNU Affero General Public License v3.0](http://www.gnu.org/licenses/agpl-3.0.txt)
