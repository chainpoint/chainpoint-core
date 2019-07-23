# First target in the Makefile is the default.
all: help

SHELL := /bin/bash

# Get the location of this makefile.
ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Get home directory of current users
CORE_DATADIR := $(shell eval printf "~$$USER")/.chainpoint/core

UID := $(shell id -u $$USER)
GID := $(shell id -g $$USER)

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

## git-pull                  : Git pull latest & submodule update
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
	export USERID=${UID} && export GROUPID=${GID} && docker-compose up -d

## up-no-build               : Startup without performing builds, rely on pull of images.
.PHONY : up-no-build
up-no-build:
	export USERID=${UID} && export GROUPID=${GID} && docker-compose up -d --no-build

## dev                       : Build and start all
.PHONY : dev
dev: build init-volumes
	export USERID=${UID} && export GROUPID=${GID} && docker-compose up -d

## dev-no-build              : Startup without performing builds, rely on pull of images.
.PHONY : dev-no-build
dev-no-build: init-volumes
	export USERID=${UID} && export GROUPID=${GID} && docker-compose up -d --no-build

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
	@rm -rf ${CORE_DATADIR}/data/postgresql
	@rm -rf ${CORE_DATADIR}/data/redis
	@rm -rf ${CORE_DATADIR}/config/node_1/data/*
	@rm -f ${CORE_DATADIR}/config/node_1/addrbook.json
	@cp ${CORE_DATADIR}/config/node_1/priv_validator_key.json ${CORE_DATADIR}/config/node_1/priv_validator.json || echo "priv_validator not found, file migration likely"
	@docker system prune --volumes -f

## init-volumes              : Create data folder with proper permissions
.PHONY : init-volumes
init-volumes:
	@mkdir -p ${CORE_DATADIR}/data/postgresql
	@mkdir -p ${CORE_DATADIR}/data/redis
	@mkdir -p ${CORE_DATADIR}/data/traefik
	@mkdir -p ${CORE_DATADIR}/config/node_1/data
	@mkdir -p ${CORE_DATADIR}/data/keys

## init                      : Create data folder with proper permissions
.PHONY : init
init: init-volumes
	@cli/scripts/install_deps.sh
	@node cli/init
	@rsync .env ${CORE_DATADIR}/.env
	@cp -rf config/traefik.toml ${CORE_DATADIR}/data/traefik/traefik.toml

## init-noninteractive       : Create data folder with proper permissions
.PHONY : init-noninteractive
init-noninteractive: init-volumes
	@node cli/init --PRIVATE_NETWORK=$(PRIVATE_NETWORK) --NETWORK=$(NETWORK) --PEERS=$(PEERS) --CORE_PUBLIC_IP_ADDRESS=$(CORE_PUBLIC_IP_ADDRESS) --BITCOIN_WIF=$(BITCOIN_WIF) --INSIGHT_API_URI=$(INSIGHT_API_URI) --INFURA_API_KEY=$(INFURA_API_KEY) --ETHERSCAN_API_KEY=$(ETHERSCAN_API_KEY)
	@rsync .env ${CORE_DATADIR}/.env
	@cp -rf config/traefik.toml ${CORE_DATADIR}/data/traefik/traefik.toml

## init-chain                : Pull down chainpoint network info
.PHONY : init-chain
init-chain:
	@bash -c "curl https://storage.googleapis.com/chp-private-testnet/genesis.json > ${CORE_DATADIR}/config/node_1/genesis.json"

## init-config               : Pull down chainpoint network config
.PHONY : init-config
init-config:
	@bash -c "curl https://storage.googleapis.com/chp-private-testnet/config.toml > ${CORE_DATADIR}/config/node_1/config.toml"

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

## deploy                    : deploys a swarm stack
deploy: init-volumes
	@rsync .env ${CORE_DATADIR}/.env
	@set -a && source ${CORE_DATADIR}/.env && export USERID=${UID} && export GROUPID=${GID} && set +a && docker stack deploy -c swarm-compose.yaml chainpoint-core

## stop                      : stops a swarm stack
stop:
	docker stack rm chainpoint-core || echo "removal in progress"

## clean-tendermint          : removes tendermint database, leaving postgres intact
clean-tendermint: stop
	@sleep 20 && rm -rf ${CORE_DATADIR}/config/node_1/data/*
	@cp ${CORE_DATADIR}/config/node_1/priv_validator_key.json ${CORE_DATADIR}/config/node_1/priv_validator.json
	docker system prune -af

## optimize-network          : increases number of sockets host can use
optimize-network:
	@sudo sysctl net.core.somaxconn=1024
	@sudo sysctl net.ipv4.tcp_fin_timeout=30
	@sudo sysctl net.ipv4.tcp_tw_reuse=1
	@sudo sysctl net.core.netdev_max_backlog=2000
	@sudo sysctl net.ipv4.tcp_max_syn_backlog=2048

## remove                    : stops, removes, and cleans a swarm
remove: stop clean
