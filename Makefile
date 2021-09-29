# First target in the Makefile is the default.
all: help

SHELL := /bin/bash

# Get the location of this makefile.
ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Get home directory of current users
HOMEDIR := $(shell eval printf "~$$USER")
CORE_DATADIR := ${HOMEDIR}/.chainpoint/core

UID := $(shell id -u $$USER)
GID := $(shell id -g $$USER)

DEP := $(shell command -v dep 2> /dev/null)
BUILD_TAGS?='tendermint'
DEV=dev
PROD=experimental
LND=autopilotrpc chainrpc invoicesrpc routerrpc signrpc signerrpc walletrpc watchtowerrpc monitoring
BUILD_FLAGS = -ldflags "-X github.com/tendermint/tendermint/version.GitCommit=`git rev-parse --short=8 HEAD`"

.PHONY : build
build:
	CGO_ENABLED=1 go build -tags "$(BUILD_TAGS) cleveldb gcc $(PROD) $(LND)" chainpoint-core.go
	echo "setting up permissions for port 80..." && sudo setcap 'cap_net_bind_service=+ep' chainpoint-core

.PHONY : install
install:
	CGO_ENABLED=1 go install -tags "$(BUILD_TAGS) cleveldb gcc $(PROD) $(LND)" chainpoint-core.go
	echo "setting up permissions for port 80..." && sudo setcap 'cap_net_bind_service=+ep' ${GOPATH}/bin/chainpoint-core

.PHONY : build-dev
build-dev:
	CGO_ENABLED=1 go build -tags "$(BUILD_TAGS) cleveldb gcc $(DEV) $(LND)" chainpoint-core.go
	echo "setting up permissions for port 80..." && sudo setcap 'cap_net_bind_service=+ep' chainpoint-core

.PHONY : install-dev
install-dev:
	CGO_ENABLED=1 go install -tags "$(BUILD_TAGS) cleveldb gcc $(DEV) $(LND)" chainpoint-core.go
	echo "setting up permissions for port 80..." && sudo setcap 'cap_net_bind_service=+ep' ${GOPATH}/bin/chainpoint-core

.PHONY : install-deps
install-deps:
	bash ./config/install_deps.sh

.PHONY : install-blocklist
install-blocklist:
	cp ./config/ip_blocklist.txt ${HOME}/.chainpoint/core/ip_blocklist.txt

.PHONY : install-daemon
install-daemon:
	envsubst < ./config/chainpoint.service.template > ./config/chainpoint.service
	sudo cp ./config/chainpoint.service /lib/systemd/system
	sudo systemctl daemon-reload
	sudo systemctl enable chainpoint

.PHONY : start-daemon
start-daemon:
	sudo systemctl start chainpoint

.PHONY : stop-daemon
stop-daemon:
	sudo systemctl stop chainpoint

.PHONY : status-daemon
status-daemon:
	sudo systemctl status chainpoint

.PHONY : log-daemon
log-daemon:
	journalctl --unit chainpoint --follow

## init-volumes              : Create data folder with proper permissions
.PHONY : init-volumes
init-volumes:
	@mkdir -p ${CORE_DATADIR}/data/keys
	@mkdir -p ${HOMEDIR}/.chainpoint/core/.lnd

## burn                      : Burn it all down and destroy the data. Start it again yourself!
.PHONY : burn
burn:
	@rm -rf ${HOMEDIR}/.chainpoint/core/.lnd

## clean-tendermint          : removes tendermint database, leaving postgres intact
clean-tendermint:
	@sleep 20 && rm -rf ${CORE_DATADIR}/data/*

## optimize-network          : increases number of sockets host can use
optimize-network:
	@sudo sysctl net.core.somaxconn=1024
	@sudo sysctl net.ipv4.tcp_fin_timeout=30
	@sudo sysctl net.ipv4.tcp_tw_reuse=1
	@sudo sysctl net.core.netdev_max_backlog=2000
	@sudo sysctl net.ipv4.tcp_max_syn_backlog=2048

