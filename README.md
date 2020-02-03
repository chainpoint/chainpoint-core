# Chainpoint Core

[![code style: prettier](https://img.shields.io/badge/code_style-prettier-ff69b4.svg?style=flat-square)](https://github.com/prettier/prettier)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Chainpoint is a protocol for anchoring data the Bitcoin blockchain. The Chainpoint Core software runs as a node on a distributed network. Cores receive hashes, aggregate these hashes into a [Merkle root](https://en.wikipedia.org/wiki/Merkle_tree), and periodically commit the root hash to the Bitcoin blockchain.

By default, Cores are members of the [Lightning Network](https://lightning.network/). Users use Lightning to pay Cores for permission to anchor a hash. Additionally, Lightning is used by new Cores to stake bitcoin to the Chainpoint Network as part of an anti-sybil mechanism. 

## Important Notice

This software is intended to be run as part of Chainpoint's Core Network. It is for operators wanting to help run the anchoring service. If you are interested in running a Chainpoint Node, or installing a copy of our command line interface please instead visit:

[https://github.com/chainpoint/chainpoint-node-src](https://github.com/chainpoint/chainpoint-node-src)

[https://github.com/chainpoint/chainpoint-cli](https://github.com/chainpoint/chainpoint-cli)

## Installing Chainpoint Core

### Requirements

An Ubuntu or MacOS system with Git, Make, and BASH are required for operation. A bash script to install all other dependencies (docker, openssl, nodejs, yarn) on Ubuntu and Mac can be found [here](https://github.com/chainpoint/chainpoint-core/blob/master/cli/scripts/install_deps.sh).

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

It _is_ possible to run Core from home, but you must have a static IP and have publicly forwarded ports 80, 443, 8080, 9000, 9735, 10009, 26656, and 26657 on your router.

### Installation

Running the following commands in BASH will download and setup the Core installation:

```
git clone https://github.com/chainpoint/chainpoint-core.git
cd chainpoint-core
make init
```

The above make command will download all other dependencies and run an interactive setup wizard. The process is further detailed in `Configuration` below.                                                                                      |

### Startup

To start up a Core node without connecting to the rest of the Chainpoint Network:

1. Run `make init` to initialize the configuration directory.

2. A `Lightning Wallet Address` will be printed in the terminal by the `make init` process. Fund this address with Bitcoin and wait for 6 confirmations.

3. Run `make deploy` to download all containers and start all services. 

If startup is successful, running `docker service logs -f chainpoint-core_abci` will show the log message `Executed block` every minute.

### Joining the Chainpoint Testnet

After running `make init` and funding the Lightning Wallet Address, you can join the public testnet as a Full Node:

1. Specify peers by adding `PEERS="087186cd1d631c5e709c4afa15a1ce218c6a28c1@3.133.119.65:26656"` to the .env file in the root project directory

2. Run `make deploy` to start Core. In order to obtain permission to submit hashes to the network, your Core will automatically stake bitcoin by opening lightning channels with the existing network validators.

### Upgrade

You can upgrade Core by running `make clean-tendermint` and `docker-compose pull`, then by redeploying with `make deploy`.

### Configuration

You will need to set up a configuration and secrets (lightning wallet) before running. `make init` will do most of the heavy lifting for you.

Chainpoint Core currently uses Docker Swarm when running in Production mode. Running `make init` will initialize a Docker Swarm node on the host machine and prompt the user for the the type of network (TESTNET or MAINNET) and public IP.
There are further settings found in the `.env.sample` and `swarm-compose.yaml` file.
These are more permanent and altering them may cause problems connecting to the public Chainpoint testnets and mainnet.
However, they may be invaluable for setting up a private Chainpoint Network with different parameters, for example by configuring more frequent bitcoin anchoring.

The following are the descriptions of the configuration parameters:

| Name                     | Type    | Location                     | Description                                                                                                                                      |
| :----------------------- | :------ | :--------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------- |
| CHAINPOINT_CORE_BASE_URI | String  | .env                         | Public URI of host machine, of the form `http://35.245.53.181`                                                                                   |
| NETWORK                  | String  | .env                         | Set to `testnet` to use Bitcoin testnet. Default is `mainnet`.                                                                     |
| NODE_ENV                 | String  | .env                         | Sets Core to use either bitcoin mainnets (`production`) or testnets (`development`). Defaults to `production`                           |
| PEERS                    | String  | .env                         | Comma-delimited list of Tendermint peer URIs of the form $ID@$IP:\$PORT, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| SEEDS                    | String  | .env                         | Comma-delimited list of Tendermint seed URIs of the form $ID@$IP:\$PORT, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| ANCHOR_INTERVAL          | String  | swarm-compose.yaml           | how often, in block time, the Core network should be anchored to Bitccoin. Default is 60.                                                        |
| HASHES_PER_MERKLE_TREE   | String  | swarm-compose.yaml           | maximum number of hashes the aggregation process will consume per aggregation interval. Default is 250000                                        |
| AGGREGATE                | Boolean | swarm-compose.yaml           | Whether to aggregate hashes and send them to the Calendar blockchain. Defaults to true                                                           |
| ANCHOR                   | Boolean | swarm-compose.yaml           | Whether to anchor the state of the Calendar to Bitcoin                                                                                           |
| LOG_FILTER               | String  | swarm-compose.yaml           | Log Verbosity. Defaults to `"main:debug,state:info,*:error"`                                                                                     |
| LOG_LEVEL                | String  | swarm-compose.yaml           | Level of detail included in Logs. Defaults to `info`       

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
| node-btc-mon-service     | Monitors the above Bitcoin TX for 6 confirmations and informs go-abci-service when complete                              | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-btc-mon-service/README.md)     |
| node-lnd-mon-service     | Monitors the lnd invoices for payment status                                                                             | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-lnd-mon-service/README.md)     |
| node-proof-gen-service   | Generates cryptographic proofs demonstrating how Chainpoint-Node data is included in the Chainpoint Calendar and Bitcoin | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-gen-service/README.md)   |
| node-proof-state-service | Stores proofs in PostgreSQL                                                                                              | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-state-service/README.md) |

## License

[Apache License Version 2.0](https://opensource.org/licenses/Apache-2.0)
