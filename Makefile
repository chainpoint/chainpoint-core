# First target in the Makefile is the default.
all: help

SHELL := /bin/bash

# Get the location of this makefile.
ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Get home directory of current users
CORE_DATADIR := $(shell eval printf "~$$USER")/.chainpoint/core

# Specify the binary dependencies
REQUIRED_BINS := docker docker-compose
$(foreach bin,$(REQUIRED_BINS),\
    $(if $(shell command -v $(bin) 2> /dev/null),$(),$(error Please install `$(bin)` first!)))

.PHONY : help
help : Makefile
	@sed -n 's/^##//p' $<

## build-config              : Copy the .env config from sample if not present
.PHONY : build-config
build-config:
	@[ ! -f ./.env ] && \
	cp .env.sample .env && \
	echo 'Copied config sample to .env' || true

## build                     : Build all
.PHONY : build
build:
	docker container prune -f
	docker-compose build

## pull                      : Pull Docker images
.PHONY : pull
pull:
	docker-compose pull

## git-pull        : Git pull latest & submodule update
.PHONY : git-pull
git-pull:
	@git pull --all
	@git submodule update --init --remote --recursive

## test-api                  : Run API test suite with Mocha
.PHONY : test-api
test-api: 
	docker-compose up --build api-test

## test-aggregator           : Run aggregator test suite
.PHONY : test-aggregator
test-aggregator:
	scripts/test.sh aggregator

## test-merkletools          : Run merkletools test suite
.PHONY : test-merkletools
test-merkletools:
	scripts/test.sh merkletools

## test-abci                 : Run abci test suite
.PHONY : test-abci
test-abci:
	scripts/test.sh abci

## test-calendar	           : Run calendar test suite
.PHONY : test-calendar
test-calendar:
	scripts/test.sh calendar

## test-util                 : Run util test suite
.PHONY : test-util
test-util:
	scripts/test.sh util

## test-rabbitmq	           : Run rabbit test suite
.PHONY : test-rabbit
test-rabbit:
	scripts/test.sh rabbit

## test-monitor              : Run monitor test suite
.PHONY : test-monitor
test-monitor:
	scripts/test.sh monitor

## test                      : Run all application tests
.PHONY : test
test: test-api test-aggregator test-merkletools test-abci test-calendar test-util test-rabbit test-monitor

## up                        : Build and start all
.PHONY : up
up: pull
	docker-compose up -d

## up-no-build               : Startup without performing builds, rely on pull of images.
.PHONY : up-no-build
up-no-build:
	docker-compose up -d --no-build

## dev                       : Build and start all
.PHONY : dev
dev: build
	docker-compose up -d

## dev-no-build              : Startup without performing builds, rely on pull of images.
.PHONY : dev-no-build
dev-no-build:
	docker-compose up -d --no-build

## down                      : Shutdown Application
.PHONY : down
down:
	docker-compose down

## ps                        : View running processes
.PHONY : ps
ps:
	docker-compose ps

## restart                   : Restart a dev mode container
.PHONY : restart
restart:
	docker-compose up -d --build $(app)

## logs                      : Tail application logs
.PHONY : logs
logs:
	docker service logs -f

## clean                     : Shutdown and destroy all local application data
.PHONY : clean
clean: down
	@sudo rm -rf ${CORE_DATADIR}/data/postgresql/*
	@sudo rm -rf ${CORE_DATADIR}/data/redis/*
	@sudo chmod 777 ${CORE_DATADIR}/data/postgresql
	@sudo chmod 777 ${CORE_DATADIR}/data/redis
	@sudo rm -rf ${CORE_DATADIR}/config/node_1/data/*
	@sudo rm -f ${CORE_DATADIR}/config/node_1/addrbook.json
	@sudo chmod 777 ${CORE_DATADIR}/config/node_1
	@sudo chmod 777 ${CORE_DATADIR}/config/node_1/*
	@sudo cp ${CORE_DATADIR}/config/node_1/priv_validator_key.json ${CORE_DATADIR}/config/node_1/priv_validator.json || echo "priv_validator not found, file migration likely"
	@sudo docker system prune --volumes -f

## init                      : Create data folder with proper permissions
.PHONY : init
init:
	@sudo mkdir -p ${CORE_DATADIR}/data
	@sudo mkdir -p ${CORE_DATADIR}/config/node_1/data
	@sudo mkdir -p ${CORE_DATADIR}/data/postgresql
	@sudo mkdir -p ${CORE_DATADIR}/data/redis
	@sudo mkdir -p ${CORE_DATADIR}/data/keys
	@sudo chmod -R 777 ${CORE_DATADIR}/data
	@sudo chmod -R 777 ${CORE_DATADIR}/config/node_1
	@docker run -it --rm -v ${CORE_DATADIR}/config/node_1:/tendermint/config  -v ${CORE_DATADIR}/config/node_1/data:/tendermint/data tendermint/tendermint init || echo "Tendermint already initialized"
	@sudo chmod 777 ${CORE_DATADIR}/config/node_1/*
	@sudo chmod 777 ${CORE_DATADIR}/config/node_1/priv_validator_key.json || echo "not yet run for the first time"
	@cp ${CORE_DATADIR}/config/node_1/priv_validator_key.json ${CORE_DATADIR}/config/node_1/priv_validator.json || echo "not yet run for the first time"
	@cli/scripts/install_deps.sh
	@node cli/init
	@sudo rsync .env ${CORE_DATADIR}/.env

## init-chain                      : Pull down chainpoint network info
.PHONY : init-chain
init-chain:
	@sudo bash -c "curl https://storage.googleapis.com/chp-private-testnet/config.toml > ${CORE_DATADIR}/config/node_1/config.toml"
	@sudo bash -c "curl https://storage.googleapis.com/chp-private-testnet/genesis.json > ${CORE_DATADIR}/config/node_1/genesis.json"

## prune                     : Shutdown and destroy all docker assets
.PHONY : prune
prune: down
	docker container prune -f
	docker image prune -f -a
	docker volume prune -f
	docker network prune -f

## prune-node-modules        : Remove the node_modules sub-directory for each service
.PHONY : prune-node-modules
prune-node-modules:
	find . -type d -name node_modules -mindepth 1 -maxdepth 2 -exec rm -rf {} \;

## burn                      : Burn it all down and destroy the data. Start it again yourself!
.PHONY : burn
burn: clean
	cli/scripts/remove_secrets.sh
	@docker swarm leave -f
	@echo ""
	@echo "****************************************************************************"
	@echo "Services stopped, and data pruned. Run 'make init' now."
	@echo "****************************************************************************"

## yarn                      : Install Node Javascript dependencies
.PHONY : yarn
yarn:
	docker run -it --rm --volume "$(PWD)":/usr/src/app --volume /var/run/docker.sock:/var/run/docker.sock --volume ~/.docker:/root/.docker --volume "$(PWD)":/wd --workdir /wd gcr.io/chainpoint-registry/github-chainpoint-chainpoint-services/node-base:latest yarn

## postgres                  : Connect to the local PostgreSQL with `psql`	
.PHONY : postgres
postgres:
	@docker-compose up-swarm -d postgres
	@sleep 6
	@docker exec -it postgres-core psql -U chainpoint

## redis                     : Connect to the local Redis with `redis-cli`
.PHONY : redis
redis:
	@docker-compose up-swarm -d redis
	@sleep 2
	@docker exec -it redis-core redis-cli

## deploy					: deploys a swarm stack
deploy:
	@rsync .env ${CORE_DATADIR}/.env
	@set -a && source ${CORE_DATADIR}/.env && set +a && docker stack deploy -c swarm-compose.yaml chainpoint-core

## stop						: stops a swarm stack
stop:
	docker stack rm chainpoint-core || echo "removal in progress"

## clean-tendermint			: removes tendermint database, leaving postgres intact
clean-tendermint: stop
	sudo rm -rf ${CORE_DATADIR}/config/node_1/data/tx_index.db
	sudo rm -rf ${CORE_DATADIR}/config/node_1/data/state.db
	sudo rm -rf ${CORE_DATADIR}/config/node_1/data/blockstore.db
	sudo rm -rf ${CORE_DATADIR}/config/node_1/data/evidence.db
	sudo rm -rf ${CORE_DATADIR}/config/node_1/data/cs.wal
	sudo rm -rf ${CORE_DATADIR}/config/node_1/data/anchor.db
	sudo rm -rf ${CORE_DATADIR}/config/node_1/data/priv_validator_state.json
	@sudo cp ${CORE_DATADIR}/config/node_1/priv_validator_key.json ${CORE_DATADIR}/config/node_1/priv_validator.json
	docker system prune -af

## remove 					: stops, removes, and cleans a swarm
remove: stop clean
