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

function abci {
    # ABCI
    echo -e "\n==Testing ABCI Application=="
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/abci -run TestABCIDeclaration
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/abci -run TestDeliverTx
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/abci -run TestABCIInfo
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/abci -run TestCommit
    make down
}

function calendar {
    # Calendar
    echo -e "\n==Testing Calendar Aggregation=="
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/calendar -run TestEmptyCalTreeGeneration
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/calendar -run TestFullCalTreeGeneration
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/calendar -run TestEmptyAnchorTreeGeneration
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/calendar -run TestFullAnchorTreeGeneration
    make down
}

function util {
    # Util
    echo -e "\n==Testing Utils=="
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/util -run TestSeededRandInt
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/util -run TestInt64ToByte
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/util -run TestByteToInt64
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/util -run TestGetEnv
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/util -run TestUUIDFromHash
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/util -run TestEncodeTx
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/util -run TestDecodeTx
    docker-compose --log-level ERROR run abci go test $PROJECT_PATH/go-abci-service/util -run TestDecodeIP
    make down
}

${COMMAND}
