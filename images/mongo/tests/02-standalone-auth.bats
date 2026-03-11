#!/usr/bin/env bats

load "helpers/common"

COMPOSE_FILE="standalone-auth.yml"

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

@test "standalone-auth: root user authenticates successfully" {
    run mongo_eval "$CONTAINER" "db.runCommand({ping:1}).ok" 27017 root rootpass123
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "standalone-auth: unauthenticated data access is rejected" {
    run mongo_eval "$CONTAINER" "db.getSiblingDB('testdb').authtest.find().toArray()"
    # Should fail or return an auth error
    [[ "$status" -ne 0 ]] || [[ "$output" == *"Unauthorized"* ]] || [[ "$output" == *"requires authentication"* ]] || [[ "$output" == *"command find requires authentication"* ]]
}

@test "standalone-auth: custom user can read/write to its database" {
    mongo_eval "$CONTAINER" \
        "db.getSiblingDB('testdb').authtest.insertOne({key:'value'})" \
        27017 testuser testpass123 testdb
    run mongo_eval "$CONTAINER" \
        "db.getSiblingDB('testdb').authtest.findOne({key:'value'}).key" \
        27017 testuser testpass123 testdb
    [ "$status" -eq 0 ]
    [[ "$output" == *"value"* ]]
}
