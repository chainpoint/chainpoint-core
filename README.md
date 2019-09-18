# Chainpoint Core

[![JavaScript Style Guide](https://img.shields.io/badge/code_style-standard-brightgreen.svg)](https://standardjs.com)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

Chainpoint is a protocol for anchoring data the Bitcoin blockchain. The Chainpoint Core software runs as a node on a distributed network. Cores receive hashes from Chainpoint Nodes, aggregate these hashes into a root hash, and periodically commit the root hash to the Bitcoin blockchain.

## Important Notice

This software is intended to be run as part of Chainpoint's Core Network. It is for operators wanting to help run the anchoring service. If you are interested in running a Chainpoint Node, or installing a copy of our command line interface please instead visit:

[https://github.com/chainpoint/chainpoint-node-src](https://github.com/chainpoint/chainpoint-node-src)

[https://github.com/chainpoint/chainpoint-cli](https://github.com/chainpoint/chainpoint-cli)

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

- `RPC address of a Bitcore Node` - Bitcoin Node with RPC enabled
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
| NETWORK                  | String  | .env                         | Set to `testnet` to use Bitcoin and Ethereum testnets. Default is `mainnet`.                                                                     |
| NODE_ENV                 | String  | .env                         | Sets Core to use either ethereum/bitcoin mainnets (`production`) or testnets (`development`). Defaults to `production`                           |
| PEERS                    | String  | .env                         | Comma-delimited list of Tendermint peer URIs of the form $ID@$IP:\$PORT, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| SEEDS                    | String  | .env                         | Comma-delimited list of Tendermint seed URIs of the form $ID@$IP:\$PORT, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
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
| node-lnd-mon-service     | Monitors the lnd invoices for payment status                                                                             | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-lnd-mon-service/README.md)     |
| node-proof-gen-service   | Generates cryptographic proofs demonstrating how Chainpoint-Node data is included in the Chainpoint Calendar and Bitcoin | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-gen-service/README.md)   |
| node-proof-state-service | Stores proofs in PostgreSQL                                                                                              | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-state-service/README.md) |

## License

[GNU Affero General Public License v3.0](http://www.gnu.org/licenses/agpl-3.0.txt)
