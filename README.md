# Chainpoint Core

[![code style: prettier](https://img.shields.io/badge/code_style-prettier-ff69b4.svg?style=flat-square)](https://github.com/prettier/prettier)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

See [Chainpoint Start](https://github.com/chainpoint/chainpoint-start) for an overview of the Chainpoint Network.

![Draft Chainpoint Architecture Diagram](https://github.com/chainpoint/chainpoint-start/blob/master/imgs/Chainpoint-Network-Overview-Diagram.png)

Chainpoint Core runs as a node on a distributed network. Cores aggregate hashes received from Gateways, maintain the Chainpoint Calendar, and periodically anchor data to the Bitcoin blockchain.

Each Core has an integrated Lightning Node running [LND](https://github.com/lightningnetwork/lnd). Cores receive `anchor fee` payments from Gateways via Lightning. The default `anchor fee` is 2 [satoshis](<https://en.bitcoin.it/wiki/Satoshi_(unit)>). Core operators can set their anchor fee to adapt to changing market conditions and compete to earn fees from Gateways.

When joining the network, new Cores automatically open Lightning channels with the existing Cores. Each channel must have a minimum capacity of 1,000,000 satoshis. This provides a measure of Sybil resistance and helps ensure Cores have sufficient liquidity to receive Lightning payments from Gateways.

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

- `>= 4GB RAM`
- `>= 2 CPU Cores`
- `100+ GB SSD`
- `Public IPv4 address`

Recommended:

- `>= 8GB RAM`
- `>= 4 CPU Cores`
- `>= 500 GB SSD`
- `Public IPv4 address`

### Quick Start

To start up a Core node and connect to the Chainpoint Network:

#### Install Dependencies

```$bash
$ sudo apt-get install make git
$ git clone https://github.com/chainpoint/chainpoint-core.git
$ cd chainpoint-core
$ make install-deps    # installs go and cleveldb
$ source ~/.bashrc     # reload path after dependency installation
$ make install         # compiles core binary
$ chainpoint-core      # Run setup
```

If you wish to daemonize the service with systemd, the following
commands are available:
```
$ make install-daemon  # install systemd if necessary
$ make start-daemon    # start systemd daemon
$ make log-daemon      # print daemon logs
```

By default the resulting binary will be placed in `~/go/bin`.

#### Configure Core

```
$ ssh user@<your_ip> #Log back into your server
$ chainpoint-core

 ██████╗██╗  ██╗ █████╗ ██╗███╗   ██╗██████╗  ██████╗ ██╗███╗   ██╗████████╗     ██████╗ ██████╗ ██████╗ ███████╗
██╔════╝██║  ██║██╔══██╗██║████╗  ██║██╔══██╗██╔═══██╗██║████╗  ██║╚══██╔══╝    ██╔════╝██╔═══██╗██╔══██╗██╔════╝
██║     ███████║███████║██║██╔██╗ ██║██████╔╝██║   ██║██║██╔██╗ ██║   ██║       ██║     ██║   ██║██████╔╝█████╗
██║     ██╔══██║██╔══██║██║██║╚██╗██║██╔═══╝ ██║   ██║██║██║╚██╗██║   ██║       ██║     ██║   ██║██╔══██╗██╔══╝
╚██████╗██║  ██║██║  ██║██║██║ ╚████║██║     ╚██████╔╝██║██║ ╚████║   ██║       ╚██████╗╚██████╔╝██║  ██║███████╗
 ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚═╝      ╚═════╝ ╚═╝╚═╝  ╚═══╝   ╚═╝        ╚═════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝


? Will this Core use Bitcoin mainnet or testnet? testnet
? Enter your Core's Public IP Address: 3.17.78.45
```

#### Auto-Initialize Lightning

```
You will need at least 3000000 Satoshis (0.03 BTC) to join the Chainpoint Network!

Initializing Lightning wallet...
Create new address for wallet...
****************************************************
Lightning initialization has completed successfully.
****************************************************
LND Wallet Password: rjcOgYehDmthuurduuriAMsr
LND Wallet Seed: absorb behind drop safe like herp derp celery galaxy wait orient sign suit castle awake gadget pass pipe sudden ethics hill choose six orphan
LND Wallet Address: tb1qfvjr20txm464fxcr0n9d4j2gkr5w4xpl55kl6u
******************************************************
You should back up this information in a secure place.
******************************************************

Please fund your Lightning address with at least [staking requirement] Satoshis (0.0X BTC) to join the Chainpoint Network`

Chainpoint Core Setup Complete. Run with chainpoint-core -config ~/.chainpoint/core/core.conf

$ chainpoint-core -config ~/.chainpoint/core/core.conf
```

The staking requirement is determined by the number of cores on the Chainpoint Network multiplied by 1,000,000 satoshis.

If startup is successful, the log output will show the log message `Executed block` every minute after the binary connects to the Chainpoint Network, and going to `<your ip>/status` in a browser will show the Core status in JSON format.

During startup, lnd and tendermint will initialize in separate background threads. During this time you may see errors or warnings related to `GetStatus` or `BlockSyncMonitor`; these can be ignored.

### Joining another Network

By default, the init process will join either the Chainpoint Testnet or Mainnet, depending on user choice. However, peering with custom networks is also possible:

1. Specify peers by adding a comma-delimited list of Tendermint URIs, such as `peers=087186cd1d631c5e709c4afa15a1ce218c6a28c1@3.133.119.65:26656` to the .conf file in the config directory (by default `~/.chainpoint/core`)

2. Run `chainpoint-core -config ~/.chainpoint/core/core.conf` to start Core. In order to obtain permission to submit hashes to the network, your Core will automatically stake bitcoin by opening lightning channels with the existing network validators.

### Upgrade

Core can be upgraded by pulling the latest main branch from this repository and recompiling with `make install`.

### Troubleshooting

If setup fails and the Lightning wallet hasn't yet been generated and funded, run `rm -rf ~/.chainpoint/core`, then run `chainpoint-core` again.

To reset the core chain state if the Lightning wallet has already been generated and funded, run `make clean-tendermint`, then `chainpoint-core` again.

For further help, [submit an issue](https://github.com/chainpoint/chainpoint-core/issues) to the Chainpoint Core repo.

### Configuration

Running chainpoint-core for the first time will perform the configuration process for you. However, you may wish to setup a custom Core or Network. To do this, you will need to set up a configuration and secrets (lightning wallet) before running.

Chainpoint Core currently uses a config file, which by default is at `~/.chainpoint/core/core.conf`. Running `chainpoint-core` without arguments for the first time will initialize this file. 
Modifying these settings may be invaluable for setting up a private Chainpoint Network with different parameters, for example by configuring more frequent bitcoin anchoring.

A full list of configuration parameters is located at abci/config.go or by running `chainpoint-core -h`.                                                                            |

## Development

We encourage anyone interested in contributing to fork this repo and submit a pull-request with desired changes. Please be sure to use eslint (npm) and gofmt (go) to check/fix any style issues.

### Build

`make build` will build a production application, provided `go` has been installed via install-deps. `make build-dev` will compile a core instance 
with tendermint and lightning node geared toward production.

### Run

After running `chainpoint-core` with test parameters, API will be accessible by default on port 80, while Tendermint is accessible on ports 26656-26657.

### Documentation

A description of the Chainpoint Network is available in the [chainpoint-start](https://github.com/chainpoint/chainpoint-start) repository.

The README for the Core application in this repo is available [here](https://github.com/chainpoint/chainpoint-core/blob/master/README.md).

## License

This is Open Source software released under [AGPLv3](./LICENSE)
