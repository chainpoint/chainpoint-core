# Chainpoint Core

[![JavaScript Style Guide](https://img.shields.io/badge/code_style-standard-brightgreen.svg)](https://standardjs.com)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

Chainpoint Core is an aggregation and anchoring service that collects hash input from Chainpoint Nodes, aggregates the hashes into a single hash, then periodically commits the data to the Bitcoin blockchain. 

## Important Notice

This software is intended to be run as the Core of the Chainpoint Network. It is for operators wanting to help run the anchoring service. If you are interested in running a Chainpoint Node, or installing a copy of our command line interface please instead visit:

[https://github.com/chainpoint/chainpoint-node-src](https://github.com/chainpoint/chainpoint-node-src)

[https://github.com/chainpoint/chainpoint-cli](https://github.com/chainpoint/chainpoint-cli)

## Quick Start

Build and start the whole system locally with `make`. Running `make help`
will display additional `Makefile` commands that are available.

```sh
git clone https://github.com/chainpoint/chainpoint-core
cd chainpoint-core
make init
make register
make deploy
```

## Getting Started

This repository contains all of the code needed to
run the full application stack locally.

You can find a script that will install all prerequisite dependencies on Mac and Linux [here](https://github.com/chainpoint/chainpoint-core/blob/master/cli/scripts/install_deps.sh).

### Setup Environment Variables

You will need to set up environment variables before building.

Running `make build-config` will copy `.env.sample` to `.env`. This file will be used by `docker-compose` to set required environment variables.

You can modify the `.env` as needed, any changes will be ignored by Git.

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
