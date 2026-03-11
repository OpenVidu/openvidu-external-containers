#!/usr/bin/env bats

load "helpers/common"

COMPOSE_FILE="standalone-noauth.yml"

setup_file() {
    compose_cleanup "$COMPOSE_FILE"
    compose_up "$COMPOSE_FILE"
    CONTAINER=$(compose_container "$COMPOSE_FILE" mongodb)
    export CONTAINER
    wait_for_mongodb "$CONTAINER"
}

teardown_file() {
    compose_cleanup "$COMPOSE_FILE"
}

@test "standalone-noauth: container is running" {
    run docker inspect -f '{{.State.Status}}' "$CONTAINER"
    [ "$status" -eq 0 ]
    [ "$output" = "running" ]
}

@test "standalone-noauth: mongosh connects without credentials" {
    run mongo_eval "$CONTAINER" "db.runCommand({ping:1}).ok"
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "standalone-noauth: can insert and query data" {
    mongo_eval "$CONTAINER" "db.getSiblingDB('testdb').items.insertOne({name:'test'})"
    run mongo_eval "$CONTAINER" "db.getSiblingDB('testdb').items.findOne({name:'test'}).name"
    [ "$status" -eq 0 ]
    [[ "$output" == *"test"* ]]
}

@test "standalone-noauth: runs as UID 1001 (non-root)" {
    run container_uid "$CONTAINER"
    [ "$status" -eq 0 ]
    [ "$output" = "1001" ]
}

@test "standalone-noauth: mongod version matches 8.0" {
    run docker exec "$CONTAINER" mongod --version
    [ "$status" -eq 0 ]
    [[ "$output" == *"v8.0"* ]]
}
