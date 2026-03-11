#!/usr/bin/env bats

load "helpers/common"

COMPOSE_FILE="docker-secrets.yml"

setup_file() {
    compose_cleanup "$COMPOSE_FILE"
    compose_up "$COMPOSE_FILE"
    CONTAINER=$(compose_container "$COMPOSE_FILE" mongodb)
    export CONTAINER
    wait_for_mongodb "$CONTAINER" 120 27017 root secretrootpass
}

teardown_file() {
    compose_cleanup "$COMPOSE_FILE"
}

@test "docker-secrets: root password loaded from file" {
    run mongo_eval "$CONTAINER" "db.runCommand({ping:1}).ok" 27017 root secretrootpass
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "docker-secrets: user password loaded from file" {
    run mongo_eval "$CONTAINER" \
        "db.getSiblingDB('secretdb').runCommand({ping:1}).ok" \
        27017 secretuser secretuserpass secretdb
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}
