# openvidu/minio

A Bitnami-compatible MinIO image built from [chainguard-forks/minio](https://github.com/chainguard-forks/minio) for OpenVidu deployments.

## Why this image?

OpenVidu deployments use MinIO as the default object storage solution for recordings and other media assets.

Since the official MinIO image has been archived and the Bitnami images have been removed from Docker Hub, this image offers a secure, actively maintained alternative. It is built from a MinIO fork that receives ongoing security patches and updates, and is designed to be a drop-in replacement for users of the original Bitnami MinIO image.

A MinIO Client (`mc`) binary is also included, reimplemented with the minimal feature set required to support the Bitnami scripts.

## Configuration

The image uses the same environment variables as the Bitnami MinIO image:

| Variable | Default | Description |
|---|---|---|
| `MINIO_ROOT_USER` | `minio` | Root user name |
| `MINIO_ROOT_PASSWORD` | `miniosecret` | Root user password |
| `MINIO_API_PORT_NUMBER` | `9000` | API port |
| `MINIO_CONSOLE_PORT_NUMBER` | `9001` | Web console port |
| `MINIO_DATA_DIR` | `/bitnami/minio/data` | Data directory |
| `MINIO_DEFAULT_BUCKETS` | _(empty)_ | Comma-separated list of buckets to create on startup |
| `MINIO_DISTRIBUTED_MODE_ENABLED` | `no` | Enable distributed mode |
| `MINIO_DISTRIBUTED_NODES` | _(empty)_ | Comma-separated list of nodes in distributed mode |
| `MINIO_SCHEME` | `http` | Scheme used by the MinIO server (`http` or `https`) |
| `MINIO_SERVER_URL` | `http://localhost:9000` | Public URL of the MinIO server |
| `MINIO_SKIP_CLIENT` | `no` | Skip MinIO Client setup on startup |
| `MINIO_FORCE_NEW_KEYS` | `no` | Force regeneration of root credentials |

All variables also support a `_FILE` suffix variant to read the value from a file (e.g. `MINIO_ROOT_PASSWORD_FILE`).

## Build

```bash
docker build \
  --build-arg MINIO_TAG=<tag> \
  -t openvidu/minio:<tag> \
  -f images/minio/Dockerfile \
  .
```

Replace `<tag>` with a valid release tag from [chainguard-forks/minio](https://github.com/chainguard-forks/minio/tags) (e.g. `RELEASE.2026-03-04T16-04-53Z`).

In CI/release workflows, tags are sourced from `versions.env` and published as:

- `openvidu/minio:${MINIO_TAG}`
- `openvidu/minio:${MINIO_TAG}-r${MINIO_BUILD_NUMBER}`
- `openvidu/minio:latest`

## Run

```bash
docker run --rm \
  -e MINIO_ROOT_USER=minio \
  -e MINIO_ROOT_PASSWORD=minio123 \
  -p 9000:9000 \
  -p 9001:9001 \
  openvidu/minio:latest
```

## Docker Compose

Three Compose files are provided for different deployment scenarios.

### Single node

```bash
docker compose -f docker-compose.yml up
```

### Distributed (four nodes, one drive each)

```bash
docker compose -f docker-compose-distributed.yml up
```

### Distributed (two nodes, two drives each)

```bash
docker compose -f docker-compose-distributed-multidrive.yml up
```
