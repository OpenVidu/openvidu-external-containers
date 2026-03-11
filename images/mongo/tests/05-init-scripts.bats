#!/usr/bin/env bats

load "helpers/common"

COMPOSE_FILE="init-scripts.yml"

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

@test "init-scripts: shell script created test_collection" {
    run mongo_eval "$CONTAINER" \
        "db.getSiblingDB('init_test').getCollectionNames().includes('test_collection')"
    [ "$status" -eq 0 ]
    [[ "$output" == *"true"* ]]
}

@test "init-scripts: JS script inserted document" {
    run mongo_eval "$CONTAINER" \
        "db.getSiblingDB('init_test').test_collection.findOne({source:'js_init_script'}).name"
    [ "$status" -eq 0 ]
    [[ "$output" == *"init_test_doc"* ]]
}

@test "init-scripts: marker file exists for idempotency" {
    run docker exec "$CONTAINER" test -f /bitnami/mongodb/.user_scripts_initialized
    [ "$status" -eq 0 ]
}
