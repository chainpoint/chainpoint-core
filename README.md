# Chainpoint Core

[![JavaScript Style Guide](https://img.shields.io/badge/code_style-standard-brightgreen.svg)](https://standardjs.com)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

Chainpoint Core is an aggregation and anchoring service that collects hash input from Chainpoint Nodes, aggregates the hashes into a single hash, then periodically commits the data to the Bitcoin blockchain. 

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

Chainpoint-Core serves as an intermediate layer between hash aggregators (Chainpoint Nodes) and Bitcoin. 
Hashes submitted by Nodes are aggregated and periodically broadcast to a Tendermint-based blockchain, the Calendar, created by consensus of all Cores. 
Every hour, a Core is elected to anchor the state of the Calendar to Bitcoin. 

To connect to an existing Chainpoint blockchain, set the PEERS environment variable in the .env file to a comma-delimited list of `<tendermint ID>@<ip>` pairs. The ID of a given Core can be found by visiting `<ip>/status`

### Configuration

Chainpoint Core requires a minimum of 2 CPU Cores and 16 GB of RAM to run. Hard disk size should be at least 500 GB for the chain data (or be expandable). 

You will need to set up a configuration and secrets (bitcoin and ethereum) before running.

Running `make init` will prompt for secrets to be stored in the docker secrets system. 
This command will also copy `.env.sample` to `.env`. The `.env` file will be used by `docker-compose` to set required environment variables.

There are further settings found in the `.env.sample` and `swarm-compose.yaml` file. 
These are more permanent and altering them may cause problems connecting to the public Chainpoint testnets and mainnet. 
However, they may be invaluable for setting up a private Chainpoint Network with different parameters, for example by configuring more frequent bitcoin anchoring or excluding the smart contract registration requirement.

The following are the descriptions of the configuration parameters:

| Name           | Type | Location |Description |
| :------------- |:-----|:---------|:-----------
| CHAINPOINT\_CORE\_BASE\_URI | String | .env | Public URI of host machine, of the form `http://35.245.53.181` |
| PRIVATE\_NETWORK | Boolean | .env | Sets Core to use pre-seeded list of Nodes instead of registry smart contract discovery. Default is false. |
| PRIVATE\_NODE\_IPS     | String | .env | Comma-delimited list of private Nodes for use with PRIVATE\_NETWORK. Default is empty string. |
| NODE\_ENV  | String | .env  | Sets Core to use either ethereum/bitcoin mainnets (`production`) or testnets (`development`) |
| ANCHOR\_INTERVAL | String | swarm-compose.yaml | how often, in block time, the Core network should be anchored to Bitccoin. Default is 60. | 
| HASHES\_PER\_MERKLE_TREE | String | swarm-compose.yaml     | maximum number of hashes the aggregation process will consume per aggregation interval. Default is 250000 | 
| PEERS | String | .env       | Comma-delimited list of Tendermint peer URIs of the form $ID@$IP:$Port, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| SEEDS | String | .env       | Comma-delimited list of Tendermint seed URIs of the form $ID@$IP:$Port, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| ETH\_INFURA\_API\_KEY | String | Docker Secrets (`make init`) | API key to use Infura ethereum web services |
| ETH\_ETHERSCAN\_API\_KEY | String | Docker Secrets (`make init`) | API key to use etherscan ethereum web services as a fallback to infura |
| ETH\_PRIVATE\_KEY | String | Docker Secrets (`make init`) | Private key for this Core's Ethereum account. |
| ECDSA\_PKPEM | String | Docker Secrets (`make init`) | Keypair used to create JWKs for Core's API auth |
| BITCOIN\_WIF | String | Docker Secrets (`make init`) | Private key for bitcoin hotwallet, used to paying anchoring fees |

## Startup

Running `make deploy` should download all contaienrs and start all services for you. 

Running `make pull` will pull docker images from the Chainpoint docker repository and start all services for you. 

Alternatively, `make build` will build all images locally.

## Build

### Build for local `docker-compose`

`make build`

### Build for GCR / DockerHub

Edit the `image:` keys for each service in the docker-compose file to reflect your desired docker repo. Run `make build`, authenticate with your docker host service, then run `docker-compose push`. 

## Development

We encourage anyone interested in contributing to fork this repo and submit a pull-request with desired changes. 

`make dev` will bring up a docker-compose instance geared toward development. API will be accessible on port 80, while Tendermint is accessible on ports 26656-26657. 

## License

[GNU Affero General Public License v3.0](http://www.gnu.org/licenses/agpl-3.0.txt)
