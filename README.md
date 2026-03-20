# openvidu-external-containers

Purpose-built container images used by OpenVidu deployments.

## Images

| Image | Description |
|---|---|
| [`mc`](images/mc/README.md) | Minimal drop-in replacement for `minio/mc` with only the commands needed by OpenVidu deployments |
| [`mimir`](images/mimir/README.md) | Alpine-based image with official `mimir` release binaries, SHA-256 verification, and non-root execution (UID/GID `1001:1001`) |
| [`mongo`](images/mongo/README.md) | Ubuntu-based Bitnami-compatible MongoDB image for OpenVidu deployments |
| [`minio`](images/minio/README.md) | MinIO server built from the chainguard-forks/minio fork at a specified tag |
