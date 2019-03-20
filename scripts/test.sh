#!/bin/bash

COMMAND="$1"
PROJECT_PATH=/go/src/github.com/chainpoint/chainpoint-core

function aggregator {
    # Aggregator
    echo -e "==Testing Aggregator=="
    docker-compose --log-level ERROR run abci go test -v $PROJECT_PATH/go-abci-service/aggregator -run Test*
    make down
}

function merkletools {
    # MerkleTools
    echo -e "\n==Testing MerkleTools=="
    docker-compose --log-level ERROR run abci go test -v $PROJECT_PATH/go-abci-service/merkletools -run Test*
    make down
}

function leader {
    # Leader Election
    echo -e "\n==Testing Leader Election=="
    docker-compose --log-level ERROR run abci go test -v $PROJECT_PATH/go-abci-service/abci -run TestLeader*
    make down
}

function abci {
    # ABCI
    echo -e "\n==Testing ABCI Application=="
    docker-compose --log-level ERROR run abci go test -v $PROJECT_PATH/go-abci-service/abci -run TestABCI*
    make down
}

function calendar {
    # Calendar
    echo -e "\n==Testing Calendar Aggregation=="
    docker-compose --log-level ERROR run abci go test -v $PROJECT_PATH/go-abci-service/calendar -run Test*
    make down
}

function util {
    # Util
    echo -e "\n==Testing Utils=="
    docker-compose --log-level ERROR run abci go test -v $PROJECT_PATH/go-abci-service/util -run Test*
    make down
}

function rabbit {
    # RabbitMQ
    echo -e "\n==Testing Rabbit=="
    docker-compose --log-level ERROR run abci go test -v $PROJECT_PATH/go-abci-service/rabbitmq -run Test*
    make down
}

function all {
    echo -e "\n==Test All==\n"
    aggregator
    merkletools
    leader
    abci
    calendar
    util
    rabbit
}

${COMMAND}
