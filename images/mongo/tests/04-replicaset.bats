#!/usr/bin/env bats

load "helpers/common"

COMPOSE_FILE="replicaset.yml"

setup_file() {
    compose_cleanup "$COMPOSE_FILE"
    compose_up "$COMPOSE_FILE"
    PRIMARY=$(compose_container "$COMPOSE_FILE" mongodb-primary)
    SECONDARY=$(compose_container "$COMPOSE_FILE" mongodb-secondary)
    ARBITER=$(compose_container "$COMPOSE_FILE" mongodb-arbiter)
    export PRIMARY SECONDARY ARBITER
    # Wait for primary to be ready
    wait_for_mongodb "$PRIMARY" 180 27017 root rootpass123
    # Wait for all members to reach their expected state (queried from primary)
    # CI runners can be slow, so allow generous timeouts for replica set convergence
    wait_for_rs_member "$PRIMARY" "mongodb-secondary" "SECONDARY" 300 root rootpass123
    wait_for_rs_member "$PRIMARY" "mongodb-arbiter" "ARBITER" 300 root rootpass123
    # Also ensure secondary container is connectable (it restarts during init)
    wait_for_mongodb "$SECONDARY" 300 27017 root rootpass123
}

teardown_file() {
    compose_cleanup "$COMPOSE_FILE"
}

@test "replicaset: primary reaches PRIMARY state" {
    run mongo_eval "$PRIMARY" \
        "rs.status().members.find(m => m.self).stateStr" \
        27017 root rootpass123
    [ "$status" -eq 0 ]
    [[ "$output" == *"PRIMARY"* ]]
}

@test "replicaset: secondary joins as SECONDARY" {
    run mongo_eval "$PRIMARY" \
        "rs.status().members.filter(m => m.stateStr === 'SECONDARY').length" \
        27017 root rootpass123
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "replicaset: arbiter joins as ARBITER" {
    run mongo_eval "$PRIMARY" \
        "rs.status().members.filter(m => m.stateStr === 'ARBITER').length" \
        27017 root rootpass123
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "replicaset: has exactly 3 members" {
    run mongo_eval "$PRIMARY" \
        "rs.status().members.length" \
        27017 root rootpass123
    [ "$status" -eq 0 ]
    [[ "$output" == *"3"* ]]
}

@test "replicaset: data replicates from primary to secondary" {
    # Write on primary
    run mongo_eval "$PRIMARY" \
        "db.getSiblingDB('repltest').items.insertOne({replicated:true})" \
        27017 root rootpass123
    [ "$status" -eq 0 ]
    # Allow replication lag
    sleep 5
    # Read from secondary (need to set secondary read preference)
    run mongo_eval "$SECONDARY" \
        "db.getMongo().setReadPref('secondary'); db.getSiblingDB('repltest').items.findOne({replicated:true}).replicated" \
        27017 root rootpass123
    [ "$status" -eq 0 ]
    [[ "$output" == *"true"* ]]
}
