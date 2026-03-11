# openvidu/mongodb

A Bitnami-compatible MongoDB image for OpenVidu deployments, built on Ubuntu.

## Why this image?

OpenVidu deployments rely on Bitnami-style startup and configuration behavior for MongoDB.

This image keeps that behavior while using Ubuntu 24.04 as base and supporting both `amd64` and `arm64`. The MongoDB runtime comes from official MongoDB packages, and the Bitnami-compatible scripts are preserved.

## Configuration

The image supports the same main environment variables used by the Bitnami MongoDB image:

| Variable | Default | Description |
|---|---|---|
| `ALLOW_EMPTY_PASSWORD` | `no` | Allows startup without `MONGODB_ROOT_PASSWORD` |
| `MONGODB_PORT_NUMBER` | `27017` | MongoDB listen port |
| `MONGODB_ROOT_USER` | `root` | Root username |
| `MONGODB_ROOT_PASSWORD` | _(empty)_ | Root password |
| `MONGODB_USERNAME` | _(empty)_ | Optional application username |
| `MONGODB_PASSWORD` | _(empty)_ | Optional application user password |
| `MONGODB_DATABASE` | _(empty)_ | Optional application database |
| `MONGODB_REPLICA_SET_MODE` | _(empty)_ | Replica set role (`primary`, `secondary`, `arbiter`) |
| `MONGODB_REPLICA_SET_NAME` | `replicaset` | Replica set name |
| `MONGODB_REPLICA_SET_KEY` | _(empty)_ | Replica set authentication key |
| `MONGODB_ADVERTISED_HOSTNAME` | _(empty)_ | Hostname advertised to peers/clients |
| `MONGODB_INITIAL_PRIMARY_HOST` | _(empty)_ | Initial primary hostname for secondary/arbiter |
| `MONGODB_EXTRA_FLAGS` | _(empty)_ | Extra `mongod` flags |

All these variables also support a `_FILE` suffix variant (for example `MONGODB_ROOT_PASSWORD_FILE`) to read values from files/secrets.

## Build

```bash
docker build \
  --build-arg YQ_VERSION=v4.52.4 \
  --build-arg WAIT_FOR_PORT_VERSION=v1.0.10 \
  --build-arg RENDER_TEMPLATE_VERSION=v1.0.9 \
  --build-arg MONGODB_MAJOR_MINOR=8.0 \
  --build-arg MONGODB_VERSION=8.0.19 \
  --build-arg MONGOSH_VERSION=2.7.0 \
  -t openvidu/mongodb:8.0.19 \
  images/mongo
```

In CI/release workflows, these values are sourced from `versions.env`.

## Run

```bash
docker run --rm \
  -e MONGODB_ROOT_PASSWORD=password123 \
  -p 27017:27017 \
  openvidu/mongodb:latest
```

## Docker Compose

From `images/mongo/`:

```bash
docker compose up
```

Replica set example:

```bash
docker compose -f docker-compose-replicaset.yml up
```

## Tests

The image includes a BATS integration suite under `images/mongo/tests` that covers standalone, auth, custom users, replica set, init scripts, persistence, custom port, docker secrets, and startup validation.
