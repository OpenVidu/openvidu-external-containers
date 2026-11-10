#!/usr/bin/env bats
# The -compat variant (--target compat), run the way OpenVidu deployments written for the
# official mongo image run it: docker-entrypoint.sh mongod with a keyFile and a replica set,
# the root user from MONGO_INITDB_ROOT_*, data in /data/db owned by UID 999.

load "helpers/common"

COMPAT_IMAGE="${MONGODB_COMPAT_IMAGE:-openvidu/mongodb:8.0-compat}"
OFFICIAL_IMAGE="${MONGODB_OFFICIAL_IMAGE:-mongo:${MONGODB_VERSION:-8.0}}"
NAME="bats-compat"

setup_file() {
    WORK="$(mktemp -d)"
    export WORK
    mkdir -p "$WORK/initdb"
    printf 'db.getSiblingDB("app").seed.insertOne({ from: "initdb" });\n' > "$WORK/initdb/seed.js"
    chmod 644 "$WORK/initdb/seed.js"
    # What the installer's setup-volumes does: directories and keyFile owned by 999
    docker run --rm -v "$WORK:/w" busybox sh -c '
        mkdir -p /w/data /w/configdb &&
        head -c 741 /dev/urandom | base64 | tr -d "\n" > /w/replica.key &&
        chown 999:999 /w/data /w/configdb /w/replica.key && chmod 600 /w/replica.key'
}

teardown_file() {
    docker rm -f "$NAME" "$NAME-official" "$NAME-bind" >/dev/null 2>&1 || true
    docker run --rm -v "$WORK:/w" busybox rm -rf /w/data /w/configdb /w/initdb /w/replica.key 2>/dev/null || true
    rmdir "$WORK" 2>/dev/null || true
}

# start <container> <image> [extra docker run args...]
start() {
    local name="$1" image="$2"
    shift 2
    docker run -d --name "$name" "$@" \
        -e MONGO_INITDB_ROOT_USERNAME=admin -e MONGO_INITDB_ROOT_PASSWORD=secret123 \
        -v "$WORK/data:/data/db" -v "$WORK/configdb:/data/configdb" -v "$WORK/replica.key:/data/replica.key" \
        -v "$WORK/initdb:/docker-entrypoint-initdb.d" \
        "$image" mongod --replSet rs0 --keyFile /data/replica.key >/dev/null
}

# rs_eval <container> <js>: authenticated mongosh, initiating the replica set on first use
rs_eval() {
    docker exec "$1" mongosh --quiet -u admin -p secret123 --authenticationDatabase admin --eval "
        try { rs.status() } catch (e) { rs.initiate({ _id: 'rs0', members: [{ _id: 0, host: '127.0.0.1:27017' }] }) }
        $2"
}

wait_primary() {
    local elapsed=0
    while [[ $elapsed -lt 120 ]]; do
        if rs_eval "$1" 'quit(rs.status().myState === 1 ? 0 : 1)' >/dev/null 2>&1; then
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    docker logs "$1" 2>&1 | tail -20 >&3
    return 1
}

@test "compat: first start creates the root user and runs the init scripts" {
    start "$NAME" "$COMPAT_IMAGE"
    wait_primary "$NAME"
    run rs_eval "$NAME" 'print(db.getSiblingDB("admin").getUser("admin").roles[0].role)'
    [[ "$output" == *"root"* ]]
    run rs_eval "$NAME" 'print(db.getSiblingDB("app").seed.countDocuments())'
    [[ "$output" == *"1"* ]]
}

@test "compat: mongod listens on every address, as the official image does" {
    local ip
    ip="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$NAME")"
    run docker exec "$NAME" mongosh --quiet --host "$ip" -u admin -p secret123 --authenticationDatabase admin \
        --eval 'print(db.runCommand({ ping: 1 }).ok)'
    [ "$status" -eq 0 ]
    [[ "$output" == *"1"* ]]
}

@test "compat: an explicit --bind_ip is kept" {
    docker run -d --name "$NAME-bind" "$COMPAT_IMAGE" mongod --bind_ip 127.0.0.1 >/dev/null
    local elapsed=0
    until docker exec "$NAME-bind" mongosh --quiet --eval 'db.runCommand({ ping: 1 }).ok' >/dev/null 2>&1; do
        [[ $elapsed -lt 60 ]] || break
        sleep 2
        elapsed=$((elapsed + 2))
    done
    run docker exec "$NAME-bind" ps -o args= -C mongod
    docker rm -f "$NAME-bind" >/dev/null
    [[ "$output" == *"--bind_ip 127.0.0.1"* ]]
    [[ "$output" != *"--bind_ip_all"* ]]
}

@test "compat: authentication is enforced" {
    run docker exec "$NAME" mongosh --quiet --eval 'db.getSiblingDB("app").x.insertOne({})'
    [ "$status" -ne 0 ]
    [[ "$output" == *"requires authentication"* ]]
}

@test "compat: mongod runs as mongodb (UID 999) and owns /data/db" {
    run docker exec "$NAME" ps -o uid= -C mongod
    [[ "$output" == *"999"* ]]
    run docker exec "$NAME" stat -c '%u:%g' /data/db
    [ "$output" = "999:999" ]
}

@test "compat: a restart keeps the data and does not initialize again" {
    rs_eval "$NAME" 'db.getSiblingDB("app").c.insertOne({ n: 1 })'
    docker restart "$NAME" >/dev/null
    wait_primary "$NAME"
    run rs_eval "$NAME" 'print(db.getSiblingDB("app").c.countDocuments())'
    [[ "$output" == *"1"* ]]
    run bash -c "docker logs $NAME 2>&1 | grep -c 'MongoDB init process complete'"
    [ "$output" = "1" ]
}

@test "compat: docker stop shuts mongod down cleanly" {
    docker stop -t 30 "$NAME" >/dev/null
    run docker inspect -f '{{.State.ExitCode}}' "$NAME"
    [ "$output" = "0" ]
    docker rm "$NAME" >/dev/null
}

@test "compat: the official image starts on the data the compat image wrote" {
    start "$NAME-official" "$OFFICIAL_IMAGE" -e GLIBC_TUNABLES=glibc.pthread.rseq=1
    wait_primary "$NAME-official"
    run rs_eval "$NAME-official" 'print(db.getSiblingDB("app").c.countDocuments())'
    [[ "$output" == *"1"* ]]
    rs_eval "$NAME-official" 'db.getSiblingDB("app").c.insertOne({ n: 2 })'
    docker stop -t 30 "$NAME-official" >/dev/null
    docker rm "$NAME-official" >/dev/null
}

@test "compat: the compat image starts on the data the official image wrote" {
    start "$NAME" "$COMPAT_IMAGE"
    wait_primary "$NAME"
    run rs_eval "$NAME" 'print(db.getSiblingDB("app").c.countDocuments())'
    [[ "$output" == *"2"* ]]
}
