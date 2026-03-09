# minio-bucket-checker

A minimal container image that replaces the [`minio/mc`](https://hub.docker.com/r/minio/mc) client in OpenVidu **HA** deployments.

## Purpose

In HA environments, certain services depend on MinIO being fully up and a target bucket existing before they can start. Previously this readiness check was implemented using `minio/mc` (`mc alias set` + `mc ls`), but that image appears to be unmaintained.

`minio-bucket-checker` replaces it entirely with a small, purpose-built Go binary that uses the official [`minio-go`](https://github.com/minio/minio-go) SDK. It polls MinIO until the server is reachable and the target bucket exists, then exits 0 so dependent services can proceed.

## Behaviour

1. Waits until MinIO is reachable (polls `ListBuckets` every 5 s)
2. Waits until the target bucket exists (polls `BucketExists` every 5 s)
3. Prints `MinIO and bucket "<name>" are ready!` and exits 0

## Environment variables

| Variable | Description | Default |
|---|---|---|
| `MINIO_HOST` | MinIO hostname or IP address | `127.0.0.1` |
| `MINIO_API_PORT_NUMBER` | MinIO API port | `9000` |
| `MINIO_ROOT_USER` | MinIO root username | — |
| `MINIO_ROOT_PASSWORD` | MinIO root password | — |
| `BUCKET_NAME` | Name of the bucket to wait for | — |
