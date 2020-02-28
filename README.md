# Chainpoint Core

[![code style: prettier](https://img.shields.io/badge/code_style-prettier-ff69b4.svg?style=flat-square)](https://github.com/prettier/prettier)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

See [Chainpoint Start](https://github.com/chainpoint/chainpoint-start) for an overview of the Chainpoint Network.

![Draft Chainpoint Architecture Diagram](https://github.com/chainpoint/chainpoint-start/blob/master/imgs/Chainpoint-Network-Overview-Diagram.png)


Chainpoint Core runs as a node on a distributed network. Cores aggregate hashes received from Gateways, maintain the Chainpoint Calendar, and periodically anchor data to the Bitcoin blockchain. 

Each Core has an integrated Lightning Node running [LND](https://github.com/lightningnetwork/lnd). Cores receive `anchor fee` payments from Gateways via Lightning. The default `anchor fee` is 2 [satoshis](https://en.bitcoin.it/wiki/Satoshi_(unit)). Core operators can set their anchor fee to adapt to changing market conditions and compete to earn fees from Gateways.

When joining the network, new Cores automatically open Lightning channels with 2/3rds of the existing Cores. Each channel must have a minimum capacity of 1,000,000 satoshis. This provides a measure of Sybil resistance and helps ensure Cores have sufficient liquidity to receive Lightning payments from Gateways. 

Once per hour, a Core is elected to anchor data to Bitcoin. As more Cores join the network, each Core anchors less frequently, thus reducing each Core’s cost of paying Bitcoin transaction fees. 

You do not need to run Chainpoint Core to use the Chainpoint protocol. Chainpoint Core is for operators that want to participate in running the anchoring service, and earn fees from Gateways.

## Installing Chainpoint Core

### Requirements

#### Software

An Ubuntu or MacOS system with Git, Make, and BASH are required for operation. A bash script to install all other dependencies (docker, openssl, nodejs, yarn) can be run from `make install-deps`.

The following tcp ports need to be open:

- Web: 80, 443
- Lightning: 8080, 9735, 10009
- Tendermint: 26656, 26657

It _is_ possible to run Core from home, but you must have a static IP and have publicly forwarded the ports above.

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

### Quick Start

To start up a Core node without connecting to the rest of the Chainpoint Network:

```$bash
$ sudo apt-get install make git
$ git clone https://github.com/chainpoint/chainpoint-core.git
$ cd chainpoint-core
$ make install-deps

Please logout and login to allow your user to use docker

$ exit #Logout of your server

$ ssh user@<your_ip> #Log back into your server
$ cd chainpoint-core
$ make init

 ██████╗██╗  ██╗ █████╗ ██╗███╗   ██╗██████╗  ██████╗ ██╗███╗   ██╗████████╗     ██████╗ ██████╗ ██████╗ ███████╗
██╔════╝██║  ██║██╔══██╗██║████╗  ██║██╔══██╗██╔═══██╗██║████╗  ██║╚══██╔══╝    ██╔════╝██╔═══██╗██╔══██╗██╔════╝
██║     ███████║███████║██║██╔██╗ ██║██████╔╝██║   ██║██║██╔██╗ ██║   ██║       ██║     ██║   ██║██████╔╝█████╗
██║     ██╔══██║██╔══██║██║██║╚██╗██║██╔═══╝ ██║   ██║██║██║╚██╗██║   ██║       ██║     ██║   ██║██╔══██╗██╔══╝
╚██████╗██║  ██║██║  ██║██║██║ ╚████║██║     ╚██████╔╝██║██║ ╚████║   ██║       ╚██████╗╚██████╔╝██║  ██║███████╗
 ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚═╝      ╚═════╝ ╚═╝╚═╝  ╚═══╝   ╚═╝        ╚═════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝


? Will this Core use Bitcoin mainnet or testnet? Testnet
? Enter your Instance's Public IP Address: 3.17.78.45

Initializing Lightning wallet...
Create new address for wallet...
Creating Docker secrets...
****************************************************
Lightning initialization has completed successfully.
****************************************************
LND Wallet Password: rjcOgYehDmthuurduuriAMsr
LND Wallet Seed: absorb behind drop safe like herp derp celery galaxy wait orient sign suit castle awake gadget pass pipe sudden ethics hill choose six orphan
LND Wallet Address: tb1qfvjr20txm464fxcr0n9d4j2gkr5w4xpl55kl6u
******************************************************
You should back up this information in a secure place.
******************************************************

Please fund the Lightning Wallet Address above with Bitcoin and wait for 6 confirmation before running 'make deploy'

$ make deploy
```

If startup is successful, running `docker service logs -f chainpoint-core_abci` will show the log message `Executed block` every minute after the docker containers download, and going to `<your ip>/status` in a browser will show the Core status in JSON format.

### Joining a Network

By default, the init process will join either the Chainpoint Testnet or Mainnet, depending on user choice. However, peering with custom networks is also possible:

1. Specify peers by adding a comma-delimited list of Tendermint URIs, such as `PEERS="087186cd1d631c5e709c4afa15a1ce218c6a28c1@3.133.119.65:26656"` to the .env file in the root project directory

2. Run `make deploy` to start Core. In order to obtain permission to submit hashes to the network, your Core will automatically stake bitcoin by opening lightning channels with the existing network validators.

### Upgrade

Core can be upgraded by running `make clean-tendermint` and `docker-compose pull`, then by redeploying with `make deploy`.

### Troubleshooting

If `make init` fails and the Lightning wallet hasn't yet been generated and funded, run `make burn`, then run `make init` again.

To reset the core chain state if the Lightning wallet has already been generated and funded, run `make clean-tendermint`, then `make init` again.

For further help, [submit an issue](https://github.com/chainpoint/chainpoint-core/issues) to the Chainpoint Core repo.

### Configuration

`make init` will perform the configuration process for you. However, you may wish to setup a custom Core or Network. To do this, you will need to set up a configuration and secrets (lightning wallet) before running.

Chainpoint Core currently uses Docker Swarm when running in Production mode. Running `make init` will initialize a Docker Swarm node on the host machine and prompt the user for the the type of network (TESTNET or MAINNET) and public IP.
There are further settings found in the `.env.sample` and `swarm-compose.yaml` file.
These are more permanent and altering them may cause problems connecting to the public Chainpoint testnets and mainnet.
However, they may be invaluable for setting up a private Chainpoint Network with different parameters, for example by configuring more frequent bitcoin anchoring.

The following are the descriptions of the configuration parameters:

| Name                     | Type    | Location           | Description                                                                                                                                      |
| :----------------------- | :------ | :----------------- | :----------------------------------------------------------------------------------------------------------------------------------------------- |
| CHAINPOINT_CORE_BASE_URI | String  | .env               | Public URI of host machine, of the form `http://35.245.53.181`                                                                                   |
| NETWORK                  | String  | .env               | Set to `testnet` to use Bitcoin testnet. Default is `mainnet`.                                                                                   |
| SUBMIT_HASH_PRICE_SAT    | String  | .env               | Price required to submit hashes to the API in satoshis                                                                                           |
| NODE_ENV                 | String  | .env               | Sets Core to use either bitcoin mainnets (`production`) or testnets (`development`). Defaults to `production`                                    |
| PEERS                    | String  | .env               | Comma-delimited list of Tendermint peer URIs of the form $ID@$IP:\$PORT, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| SEEDS                    | String  | .env               | Comma-delimited list of Tendermint seed URIs of the form $ID@$IP:\$PORT, such as `73d315d7c92e60df6aa92632259def61cace59de@35.245.53.181:26656`. |
| ANCHOR_INTERVAL          | String  | swarm-compose.yaml | how often, in block time, the Core network should be anchored to Bitccoin. Default is 60.                                                        |
| AGGREGATOR_WHITELIST     | String  | swarm-compose.yaml | Comma-delimited list of IPs that are permitted to use Core's API without following the LSAT auth flow                                            |
| HASHES_PER_MERKLE_TREE   | String  | swarm-compose.yaml | maximum number of hashes the aggregation process will consume per aggregation interval. Default is 250000                                        |
| AGGREGATE                | Boolean | swarm-compose.yaml | Whether to aggregate hashes and send them to the Calendar blockchain. Defaults to true                                                           |
| ANCHOR                   | Boolean | swarm-compose.yaml | Whether to anchor the state of the Calendar to Bitcoin                                                                                           |
| LOG_FILTER               | String  | swarm-compose.yaml | Log Verbosity. Defaults to `"main:debug,state:info,*:error"`                                                                                     |
| LOG_LEVEL                | String  | swarm-compose.yaml | Level of detail included in Logs. Defaults to `info`                                                                                             |

## Development

We encourage anyone interested in contributing to fork this repo and submit a pull-request with desired changes. Please be sure to use eslint (npm) and gofmt (go) to check/fix any style issues.

### Build

`make build` will build and tag local docker images for each of the micro-services in this Repo.

### Run

`make dev` will bring up a docker-compose instance geared toward development. API will be accessible on port 80, while Tendermint is accessible on ports 26656-26657.

### Documentation

READMEs for each Core micro-service are available:

| Service                  | Description                                                                                                           | Readme                                                                                                 |
| :----------------------- | :-------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------- |
| go-abci-service          | Runs the Tendermint blockchain service and coordinates all Core activity                                              | [README](https://github.com/chainpoint/chainpoint-core/blob/master/go-abci-service/README.md)          |
| node-api-service         | Web API for interacting with Chainpoint-Gateways                                                                      | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-api-service/README.md)         |
| node-btc-mon-service     | Monitors the above Bitcoin TX for 6 confirmations and informs go-abci-service when complete                           | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-btc-mon-service/README.md)     |
| node-lnd-mon-service     | Monitors the lnd invoices for payment status                                                                          | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-lnd-mon-service/README.md)     |
| node-proof-gen-service   | Generates cryptographic proofs showing how Chainpoint-Gatewat data is included in the Chainpoint Calendar and Bitcoin | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-gen-service/README.md)   |
| node-proof-state-service | Stores proofs in PostgreSQL                                                                                           | [README](https://github.com/chainpoint/chainpoint-core/blob/master/node-proof-state-service/README.md) |

## License

[Apache License Version 2.0](https://opensource.org/licenses/Apache-2.0)
