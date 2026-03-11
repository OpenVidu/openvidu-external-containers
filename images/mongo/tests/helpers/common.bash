#!/usr/bin/env bash
# Shared helpers for MongoDB BATS tests

# Default image; override via MONGODB_IMAGE env var
export MONGODB_IMAGE="${MONGODB_IMAGE:-openvidu/mongodb:8.0}"

# Absolute path to tests/ directory
export TESTS_DIR
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ── Docker Compose helpers ──────────────────────────────────────────

# Start services from a compose file inside tests/compose/
# Usage: compose_up <compose-file-basename> [extra docker compose args...]
compose_up() {
    local file="$1"; shift
    docker compose -f "${TESTS_DIR}/compose/${file}" -p "bats-${file%.yml}" up -d "$@"
}

# Tear down services and volumes for a compose file
compose_cleanup() {
    local file="$1"; shift
    docker compose -f "${TESTS_DIR}/compose/${file}" -p "bats-${file%.yml}" down -v --remove-orphans "$@" 2>/dev/null || true
}

# Get the container ID for a service
compose_container() {
    local file="$1" service="$2"
    docker compose -f "${TESTS_DIR}/compose/${file}" -p "bats-${file%.yml}" ps -q "$service"
}

# ── Wait helpers ────────────────────────────────────────────────────

# Wait for MongoDB to be ready inside a container.
# Usage: wait_for_mongodb <container_id> [timeout_seconds] [port] [user] [password] [auth_db]
wait_for_mongodb() {
    local container="$1"
    local timeout="${2:-120}"
    local port="${3:-27017}"
    local user="${4:-}"
    local password="${5:-}"
    local auth_db="${6:-admin}"
    local elapsed=0

    local auth_args=""
    if [[ -n "$user" && -n "$password" ]]; then
        auth_args="-u '${user}' -p '${password}' --authenticationDatabase '${auth_db}'"
    fi

    local consecutive_ok=0
    while [[ $elapsed -lt $timeout ]]; do
        if docker exec "$container" bash -c \
            ". /opt/bitnami/scripts/mongodb-env.sh && mongosh --quiet --port ${port} ${auth_args} --eval 'db.runCommand({ping:1}).ok'" 2>/dev/null | grep -q '1'; then
            consecutive_ok=$((consecutive_ok + 1))
            # Require 3 consecutive successes to avoid catching the setup-phase mongod
            if [[ $consecutive_ok -ge 3 ]]; then
                return 0
            fi
        else
            consecutive_ok=0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    echo "Timed out waiting for MongoDB on container ${container} (${timeout}s)" >&2
    return 1
}

# Wait for a replica set member to reach a specific state, queried from the primary.
# Usage: wait_for_rs_member <primary_container> <member_hostname> <expected_state> [timeout] [user] [password]
wait_for_rs_member() {
    local primary="$1"
    local member_host="$2"
    local expected_state="$3"
    local timeout="${4:-120}"
    local user="${5:-}"
    local password="${6:-}"
    local elapsed=0

    while [[ $elapsed -lt $timeout ]]; do
        local state
        state=$(mongo_eval "$primary" \
            "var m = rs.status().members.find(m => m.name.startsWith('${member_host}')); m ? m.stateStr : 'UNKNOWN'" \
            27017 "$user" "$password" 2>/dev/null || true)
        if [[ "$state" == *"$expected_state"* ]]; then
            return 0
        fi
        sleep 3
        elapsed=$((elapsed + 3))
    done
    echo "Timed out waiting for ${member_host} to reach ${expected_state} (${timeout}s)" >&2
    return 1
}

# ── Mongo evaluation helpers ───────────────────────────────────────

# Run a mongosh expression inside a container and return stdout.
# Usage: mongo_eval <container> <js_expression> [port] [user] [password] [auth_db]
mongo_eval() {
    local container="$1"
    local expression="$2"
    local port="${3:-27017}"
    local user="${4:-}"
    local password="${5:-}"
    local auth_db="${6:-admin}"

    local auth_args=""
    if [[ -n "$user" && -n "$password" ]]; then
        auth_args="-u '${user}' -p '${password}' --authenticationDatabase '${auth_db}'"
    fi

    docker exec "$container" bash -c \
        ". /opt/bitnami/scripts/mongodb-env.sh && mongosh --quiet --port ${port} ${auth_args} --eval \"${expression}\""
}

# ── Container inspection helpers ───────────────────────────────────

# Get the UID running the main process inside a container
container_uid() {
    docker exec "$1" id -u
}

# Dump container logs (useful on failure)
dump_logs() {
    local file="$1" service="$2"
    echo "=== Logs for ${service} ==="
    docker compose -f "${TESTS_DIR}/compose/${file}" -p "bats-${file%.yml}" logs "$service" 2>&1 || true
}
