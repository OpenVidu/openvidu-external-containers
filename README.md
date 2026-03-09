# openvidu-external-containers

Purpose-built container images used by OpenVidu deployments.

## Images

| Image | Description |
|---|---|
| [`minio-bucket-checker`](images/minio-bucket-checker/README.md) | Replaces `minio/mc` in HA deployments. It waits until MinIO is reachable and a target bucket exists, then exits 0 |
| [`minio`](images/minio/README.md) | MinIO server built from the chainguard-forks/minio fork at a specified tag |
