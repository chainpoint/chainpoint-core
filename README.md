# Chainpoint Core

[![JavaScript Style Guide](https://img.shields.io/badge/code_style-standard-brightgreen.svg)](https://standardjs.com)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

Chainpoint Core is a data integrity service which collects hash input from [Chainpoint Nodes](https://github.com/chainpoint/chainpoint-node-src) via a web API, aggregates the hashes into a single root hash, then periodically commits the root hash to the Bitcoin blockchain as a form of distributed digital notarization.

## Important Notice

This software is intended to be run as part of Chainpoint's Core Network. It is for operators wanting to help run the anchoring service. If you are interested in running a Chainpoint Node, or installing a copy of our command line interface please instead visit:

[https://github.com/chainpoint/chainpoint-node-src](https://github.com/chainpoint/chainpoint-node-src)

[https://github.com/chainpoint/chainpoint-cli](https://github.com/chainpoint/chainpoint-cli)

## Introduction to Chainpoint

Chainpoint is a protocol for digitally notarizing data using the Bitcoin blockchain. It makes the process of anchoring data fingerprints (hashes) to Bitcoin more cost-effective by creating an intermediate, decentralized network between the user and the Bitcoin blockchain.

The first tier is the Chainpoint Node, which aggregates user submissions into a single datapoint every minute. This datapoint is then submitted to the second tier, the Chainpoint Core Network. The Core Network consists of many Cores running in concert to create the Calendar, a Tendermint-based blockchain. Every hour, a random Core anchors the state of the Calendar to Bitcoin. The more Cores there are, the less often a given Core is selected to anchor, which reduces the burden of paying Bitcoin fees.

After the Cores anchor to Bitcoin, the Chainpoint Nodes retrieve the result and use it to construct a cryptographic proof. This proof shows that the Bitcoin blockchain contains a hash of the user's data. 

Users can install a release of Chainpoint-CLI to submit data to a Chainpoint Node and retrieve a Chainpoint Proof.

## What is Chainpoint Core?

Chainpoint Core forms the backbone of the Chainpoint Network and is operated by 'operators'- dedicated users and organizations. By running a Core and joining the Chainpoint Network, an operator helps defray the cost of transactions fees associated with anchoring data to Bitcoin.
Additionally, the decentralized nature of Core adds significant resiliency to the Network. If a Core drops offline, the remaining Cores continue to elect a leader to anchor the Calendar to Bitcoin.

Core uses [Tendermint](https://github.com/tendermint/tendermint) to communicate with other Cores and generate the Calendar blockchain. Because Tendermint uses a Validator-based model, generating new blocks through _mining_ is not necessary.
Instead, certain trustworthy, secure Cores (Validators) agree on new Calendar blocks. It should be noted that it is _not_ necessary for a Core operator to be a Validator in order to anchor to Bitcoin. You only need a wallet with some Bitcoin and the instructions below!

## Installing Chainpoint Core

### Requirements

#### Hardware

Chainpoint Core has been tested with a couple of different hardware configurations.

Minimum:

- `>= 8GB RAM`
- `>= 2 CPU Cores`
- `256+ GB SSD`
- `Public IPv4 address`

Recommended:

- `>= 16GB RAM`
- `>= 4 CPU Cores`
- `>= 500 GB SSD`
- `Public IPv4 address`
- `High-performance (1 Gbps+) Cloud Provider Networking`

It _is_ possible to run Core from home, but you must have a static IP and have publicly forwarded ports 80, 26656, and 26657 on your router.

#### Software

At minimum, the following software is required for any installation of Core:

- `*Nix-based OS (Ubuntu Linux and MacOS have been tested)`
- `BASH`
- `Git`

Provided BASH is installed, a script to install all other dependencies (make, openssl, nodejs, yarn) on Ubuntu and Mac can be found [here](https://github.com/chainpoint/chainpoint-core/blob/master/cli/scripts/install_deps.sh).

#### External Services

Core requires a few external services to facilitate communication with the Bitcoin and Ethereum blockchains. You will need:

- `IP address of a Bitcore Node` - Bitcoin Node running the Insight-API
- `Infura API Key` - generated from infura.io
- `Etherscan API key`
- `Bitcoin WIF`- Bitcoin Wallet Import Format (WIF) string in Base58.

The Bitcoin WIF is the private key of your _Hot Wallet_, which is used to pay for Anchoring fees. Do not use your main Bitcoin wallet here!

### Installation

Running the following commands in BASH will download and setup the Core installation:

```
git clone https://github.com/chainpoint/chainpoint-core.git
cd chainpoint-core
make init
```

The above make command will download all other dependencies and run an interactive setup wizard. The process is detailed in `Configuration` below.

### Configuration

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
| PEERS                    | String  | .env                         | Comma-delimited list of Tendermint peer URIs of the form $ID@$IP:\$PORT, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| SEEDS                    | String  | .env                         | Comma-delimited list of Tendermint seed URIs of the form $ID@$IP:\$PORT, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| ETH_INFURA_API_KEY       | String  | Docker Secrets (`make init`) | API key to use Infura ethereum web services                                                                                                      |
| ETH_ETHERSCAN_API_KEY    | String  | Docker Secrets (`make init`) | API key to use etherscan ethereum web services as a fallback to infura                                                                           |
| ETH_PRIVATE_KEY          | String  | Docker Secrets (`make init`) | Private key for this Core's Ethereum account.                                                                                                    |
| ECDSA_PKPEM              | String  | Docker Secrets (`make init`) | Keypair used to create JWKs for Core's API auth                                                                                                  |
| BITCOIN_WIF              | String  | Docker Secrets (`make init`) | Private key for bitcoin hotwallet, used to paying anchoring fees                                                                                 |
| ANCHOR_INTERVAL          | String  | swarm-compose.yaml           | how often, in block time, the Core network should be anchored to Bitccoin. Default is 60.                                                        |
| HASHES_PER_MERKLE_TREE   | String  | swarm-compose.yaml           | maximum number of hashes the aggregation process will consume per aggregation interval. Default is 250000                                        |
| AGGREGATE                | Boolean | swarm-compose.yaml           | Whether to aggregate hashes and send them to the Calendar blockchain. Defaults to true                                                           |
| ANCHOR                   | Boolean | swarm-compose.yaml           | Whether to anchor the state of the Calendar to Bitcoin                                                                                           |
| LOG_FILTER               | String  | swarm-compose.yaml           | Log Verbosity. Defaults to `"main:debug,state:info,*:error"`                                                                                     |
| LOG_LEVEL                | String  | swarm-compose.yaml           | Level of detail included in Logs. Defaults to `info`                                                                                             |

### Startup

To start up a Core node without connecting to the rest of the Chainpoint Network:

1. Run `make init` to initialize the configuration directory

2. Run `make register` to submit your Core as a candidate to join the Chainpoint Network

3. Run `make deploy` to download all containers and start all services.

If startup is successful, running `docker service logs -f chainpoint-core_abci` will show the log message `Executed block` every minute.

### Joining the Chainpoint Testnet

After running `make init`, you can join the public testnet as a Full Node:

1. Run `make init-chain` to download the testnet genesis and config files

2. Specify peers by adding `PEERS="4350130c60da6c4e443d9fe4abb9c4129b82a651@35.245.53.181:26656,50cf252391ff609a1de6c2a146c4fdfcff80dc41@35.188.238.186:26656"` to the .env file in the root project directory

3. Run `make deploy` to copy the .env file to the config directory and deploy Core.

### Upgrade

You can upgrade Core by running `make clean-tendermint` and `docker-compose pull`, then by redeploying with `make deploy`.

## Development

We encourage anyone interested in contributing to fork this repo and submit a pull-request with desired changes. Please be sure to use eslint (npm) and gofmt (go) to check/fix any style issues.

### Build

`make build` will build and tag local docker images for each of the micro-services in this Repo.

### Run

`make dev` will bring up a docker-compose instance geared toward development. API will be accessible on port 80, while Tendermint is accessible on ports 26656-26657.

### Documentation

READMEs for each Core micro-service are available:

| Service                  | Description                                                                                                              | Readme                                                                                                 |
| :----------------------- | :----------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------- |
| go-abci-service          | Runs the Tendermint blockchain service and coordinates all Core activity                                                 | [README](https://github.com/chainpoint/chainpoint-core/blob/master/go-abci-service/README.md)          |
| node-api-service         | Web API for interacting with Chainpoint-Nodes                                                                            | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-api-service/README.md)         |
| node-btc-tx-service      | Transmits a Merkle Root to Bitcoin and returns the Bitcoin TX ID                                                         | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-btc-tx-service/README.md)      |
| node-btc-mon-service     | Monitors the above Bitcoin TX for 6 confirmations and informs go-abci-service when complete                              | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-btc-mon-service/README.md)     |
| node-proof-gen-service   | Generates cryptographic proofs demonstrating how Chainpoint-Node data is included in the Chainpoint Calendar and Bitcoin | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-gen-service/README.md)   |
| node-proof-state-service | Stores proofs in PostgreSQL                                                                                              | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-state-service/README.md) |

## License

[GNU Affero General Public License v3.0](http://www.gnu.org/licenses/agpl-3.0.txt)
