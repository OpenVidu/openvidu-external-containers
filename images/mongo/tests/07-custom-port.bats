#!/usr/bin/env bats

load "helpers/common"

COMPOSE_FILE="standalone-custom-port.yml"

setup_file() {
    compose_cleanup "$COMPOSE_FILE"
    compose_up "$COMPOSE_FILE"
    CONTAINER=$(compose_container "$COMPOSE_FILE" mongodb)
    export CONTAINER
    wait_for_mongodb "$CONTAINER" 120 27117
}

teardown_file() {
    compose_cleanup "$COMPOSE_FILE"
}

@test "custom-port: MongoDB listens on port 27117" {
    run mongo_eval "$CONTAINER" "db.runCommand({ping:1}).ok" 27117
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "custom-port: MongoDB does NOT listen on default 27017" {
    run mongo_eval "$CONTAINER" "db.runCommand({ping:1}).ok" 27017
    [ "$status" -ne 0 ]
}
