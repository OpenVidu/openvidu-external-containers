#!/usr/bin/env bats

load "helpers/common"

COMPOSE_FILE="standalone-custom-users.yml"

setup_file() {
    compose_cleanup "$COMPOSE_FILE"
    compose_up "$COMPOSE_FILE"
    CONTAINER=$(compose_container "$COMPOSE_FILE" mongodb)
    export CONTAINER
    wait_for_mongodb "$CONTAINER" 120 27017 root rootpass123
}

teardown_file() {
    compose_cleanup "$COMPOSE_FILE"
}

@test "custom-users: alice authenticates to alicedb" {
    run mongo_eval "$CONTAINER" "db.runCommand({ping:1}).ok" 27017 alice alicepass alicedb
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "custom-users: bob authenticates to bobdb" {
    run mongo_eval "$CONTAINER" "db.runCommand({ping:1}).ok" 27017 bob bobpass bobdb
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "custom-users: charlie authenticates to charliedb" {
    run mongo_eval "$CONTAINER" "db.runCommand({ping:1}).ok" 27017 charlie charliepass charliedb
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "custom-users: alice can insert and query in alicedb" {
    mongo_eval "$CONTAINER" \
        "db.getSiblingDB('alicedb').docs.insertOne({owner:'alice'})" \
        27017 alice alicepass alicedb
    run mongo_eval "$CONTAINER" \
        "db.getSiblingDB('alicedb').docs.findOne({owner:'alice'}).owner" \
        27017 alice alicepass alicedb
    [ "$status" -eq 0 ]
    [[ "$output" == *"alice"* ]]
}

@test "custom-users: bob can insert and query in bobdb" {
    mongo_eval "$CONTAINER" \
        "db.getSiblingDB('bobdb').docs.insertOne({owner:'bob'})" \
        27017 bob bobpass bobdb
    run mongo_eval "$CONTAINER" \
        "db.getSiblingDB('bobdb').docs.findOne({owner:'bob'}).owner" \
        27017 bob bobpass bobdb
    [ "$status" -eq 0 ]
    [[ "$output" == *"bob"* ]]
}
