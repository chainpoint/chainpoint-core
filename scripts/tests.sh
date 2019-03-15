#!/bin/bash

COMMAND="$1"
PROJECT_PATH=/go/src/github.com/chainpoint/chainpoint-core

function aggregator {
    # Aggregator
    echo -e "==Testing Aggregator=="
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/aggregator/aggregator.go $PROJECT_PATH/go-abci-service/aggregator/aggregator_test.go
    make down
}

function merkletools {
    # MerkleTools
    echo -e "\n==Testing MerkleTools=="
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/merkletools/merkletools.go $PROJECT_PATH/go-abci-service/merkletools/merkletools_test.go
    make down
}

function leader {
    # Leader Election
    echo -e "\n==Testing Leader Election=="
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/abci -run TestLeaderElectionLeader
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/abci -run TestLeaderElectionNotLeader
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/abci -run TestLeaderElectionSingleCore
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/abci -run TestLeaderElectionCatchingUp
    make down
}

${COMMAND}
