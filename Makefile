# First target in the Makefile is the default.
all: help

# Get the location of this makefile.
ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Specify the binary dependencies
REQUIRED_BINS := docker docker-compose
$(foreach bin,$(REQUIRED_BINS),\
    $(if $(shell command -v $(bin) 2> /dev/null),$(),$(error Please install `$(bin)` first!)))

.PHONY : help
help : Makefile
	@sed -n 's/^##//p' $<

## cockroach                 : Local CockroachDB console
.PHONY : cockroachdb
cockroach:
	./bin/cockroach -d chainpoint sql

## cockroachdb-reset         : Bring the system down, delete CockroachDB data, setup DB as needed, and start cluster
.PHONY : cockroachdb-reset
cockroachdb-reset: down
	./bin/cockroach-setup -d

## cockroachdb-setup         : Initialize CockroachDB
.PHONY : cockroachdb-setup
cockroachdb-setup:
	./bin/cockroach-setup

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

## test-api                  : Run API test suite with Mocha
.PHONY : test-api
test-api: cockroachdb-setup
	docker-compose up --build api-test

## test-aggregator           : Run aggregator test suite with Mocha
.PHONY : test-aggregator
test-aggregator:
	docker-compose up --build aggregator-test

## test                      : Run all application tests
.PHONY : test
test: test-api test-aggregator

## up                        : Build and start all
.PHONY : up
up: build cockroachdb-setup
	docker-compose up -d --build

## up-no-build               : Startup without performing builds, rely on pull of images.
.PHONY : up-no-build
up-no-build: pull cockroachdb-setup
	docker-compose up -d --no-build

## down                      : Shutdown Application
.PHONY : down
down:
	docker-compose down

## ps                        : View running processes
.PHONY : ps
ps:
	docker-compose ps

## logs                      : Tail application logs
.PHONY : logs
logs:
	docker-compose logs -f

## clean                     : Shutdown and destroy all local application data
.PHONY : clean
clean: down
	@rm -rf ./data/*

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
	find . -type d -name node_modules -mindepth 1 -maxdepth 1 -exec rm -rf {} \;

## burn                      : Burn it all down and destroy the data. Start it again yourself!
.PHONY : burn
burn: clean prune
	@echo ""
	@echo "****************************************************************************"
	@echo "Services stopped, and data pruned. Run 'make up' or 'make up-no-build' now."
	@echo "****************************************************************************"

## yarn                      : Install Node Javascript dependencies
.PHONY : yarn
yarn:
	docker run -it --rm --volume "$(PWD)":/usr/src/app --volume /var/run/docker.sock:/var/run/docker.sock --volume ~/.docker:/root/.docker --volume "$(PWD)":/wd --workdir /wd gcr.io/chainpoint-registry/github-chainpoint-chainpoint-services/node-base:latest yarn

## redis                     : Connect to the local Redis with `redis-cli`
.PHONY : redis
redis:
	@docker-compose up -d redis
	@sleep 2
	@docker exec -it redis-core redis-cli
