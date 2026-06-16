# openvidu-external-containers

Purpose-built container images used by OpenVidu deployments.

## Images

| Image | Description |
|---|---|
| [`loki`](images/loki/README.md) | `grafana/loki` image with `/bin/sh` and `grep` for OpenVidu deployment scripts |
| [`mc`](images/mc/README.md) | Minimal drop-in replacement for `minio/mc` with only the commands needed by OpenVidu deployments |
| [`mimir`](images/mimir/README.md) | `grafana/mimir` image with `/bin/sh` and `grep` for OpenVidu deployment scripts |
| [`mongo`](images/mongo/README.md) | Ubuntu-based Bitnami-compatible MongoDB image for OpenVidu deployments |
| [`minio`](images/minio/README.md) | MinIO server built from the chainguard-forks/minio fork at a specified tag |
