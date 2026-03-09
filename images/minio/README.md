# minio

A MinIO server image built from the [chainguard-forks/minio](https://github.com/chainguard-forks/minio) fork.

## Build

```bash
docker build --build-arg MINIO_TAG=<tag> -t openvidu/minio:<tag> images/minio/
```

Replace `<tag>` with a valid tag from the [chainguard-forks/minio](https://github.com/chainguard-forks/minio) repository (e.g. `RELEASE.2024-01-01T00-00-00Z`).

## Run

```bash
docker run --rm \
  -e MINIO_ROOT_USER=minio \
  -e MINIO_ROOT_PASSWORD=minio123 \
  -p 9000:9000 \
  -p 9001:9001 \
  openvidu/minio:<tag>
```
