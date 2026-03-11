#!/usr/bin/env bats

load "helpers/common"

COMPOSE_FILE="standalone-auth.yml"
PROJECT="bats-persistence"

setup_file() {
    # Clean up any previous run
    docker compose -f "${TESTS_DIR}/compose/${COMPOSE_FILE}" -p "$PROJECT" down -v --remove-orphans 2>/dev/null || true

    # First start: insert data
    docker compose -f "${TESTS_DIR}/compose/${COMPOSE_FILE}" -p "$PROJECT" up -d
    CONTAINER=$(docker compose -f "${TESTS_DIR}/compose/${COMPOSE_FILE}" -p "$PROJECT" ps -q mongodb)
    export CONTAINER
    wait_for_mongodb "$CONTAINER" 120 27017 root rootpass123

    mongo_eval "$CONTAINER" \
        "db.getSiblingDB('testdb').persist.insertOne({survived:true})" \
        27017 root rootpass123

    # Stop (not down - keep volumes)
    docker compose -f "${TESTS_DIR}/compose/${COMPOSE_FILE}" -p "$PROJECT" stop

    # Second start
    docker compose -f "${TESTS_DIR}/compose/${COMPOSE_FILE}" -p "$PROJECT" start
    CONTAINER=$(docker compose -f "${TESTS_DIR}/compose/${COMPOSE_FILE}" -p "$PROJECT" ps -q mongodb)
    export CONTAINER
    wait_for_mongodb "$CONTAINER" 120 27017 root rootpass123
}

teardown_file() {
    docker compose -f "${TESTS_DIR}/compose/${COMPOSE_FILE}" -p "$PROJECT" down -v --remove-orphans 2>/dev/null || true
}

@test "persistence: data survives stop/start" {
    run mongo_eval "$CONTAINER" \
        "db.getSiblingDB('testdb').persist.findOne({survived:true}).survived" \
        27017 root rootpass123
    [ "$status" -eq 0 ]
    [[ "$output" == *"true"* ]]
}

@test "persistence: auth still works after restart" {
    run mongo_eval "$CONTAINER" "db.runCommand({ping:1}).ok" 27017 root rootpass123
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "persistence: logs show persisted data message on second start" {
    run docker compose -f "${TESTS_DIR}/compose/${COMPOSE_FILE}" -p "$PROJECT" logs mongodb
    [ "$status" -eq 0 ]
    [[ "$output" == *"Deploying MongoDB with persisted data"* ]]
}
