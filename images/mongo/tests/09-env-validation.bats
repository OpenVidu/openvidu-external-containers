#!/usr/bin/env bats

load "helpers/common"

# These tests use `docker run --rm` directly (no compose) to test startup
# validation failures. The container should exit non-zero with an error message.

@test "env-validation: fails without ALLOW_EMPTY_PASSWORD or root password" {
    run docker run --rm \
        -e MONGODB_USERNAME=testuser \
        -e MONGODB_PASSWORD=testpass \
        -e MONGODB_DATABASE=testdb \
        "${MONGODB_IMAGE}" \
        /opt/bitnami/scripts/mongodb/run.sh
    [ "$status" -ne 0 ]
    [[ "$output" == *"MONGODB_ROOT_PASSWORD"* ]]
}

@test "env-validation: fails with secondary mode but no INITIAL_PRIMARY_HOST" {
    run docker run --rm \
        -e MONGODB_REPLICA_SET_MODE=secondary \
        -e MONGODB_INITIAL_PRIMARY_ROOT_PASSWORD=rootpass \
        -e MONGODB_REPLICA_SET_KEY=replicasetkey123 \
        "${MONGODB_IMAGE}" \
        /opt/bitnami/scripts/mongodb/run.sh
    [ "$status" -ne 0 ]
    [[ "$output" == *"MONGODB_INITIAL_PRIMARY_HOST"* ]]
}

@test "env-validation: fails with replica set key shorter than 5 chars" {
    run docker run --rm \
        -e MONGODB_REPLICA_SET_MODE=primary \
        -e MONGODB_ROOT_PASSWORD=rootpass \
        -e MONGODB_REPLICA_SET_KEY=abc \
        "${MONGODB_IMAGE}" \
        /opt/bitnami/scripts/mongodb/run.sh
    [ "$status" -ne 0 ]
    [[ "$output" == *"at least"*"5"* ]]
}
