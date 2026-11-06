# openvidu/grafana-mimir

`grafana/mimir` image with `/bin/sh` and `grep` for OpenVidu deployment scripts.

## Why this image?

The official `grafana/mimir` image is distroless — it has no shell. OpenVidu's Mimir entrypoint script uses `/bin/sh` for conditional checks (`grep`, shell operators) before starting the mimir process.

This image provides:
- Official `mimir` binary from `grafana/mimir`
- `/bin/sh` (busybox, statically linked)
- `grep` (busybox, statically linked)
- Non-root user with UID/GID `1001:1001`

## Build

```bash
docker build \
  --build-arg MIMIR_TAG=<VERSION> \
  --build-arg MIMIR_DIGEST=<DIGEST> \
  -t openvidu/grafana-mimir:<VERSION> \
  -f images/mimir/Dockerfile \
  .
```

Where `<VERSION>` is the desired Mimir version. `<DIGEST>` (`sha256:...`) pins the base image and is optional: drop that line to build from the tag alone.

In CI/release workflows, tags are sourced from `versions.env` and published as:

- `openvidu/grafana-mimir:${MIMIR_TAG}`
- `openvidu/grafana-mimir:${MIMIR_TAG}-r<N>`, where `<N>` is the build revision: `0` for the first build of a version, then incremented on every rebuild (for example after a base image update)
- `openvidu/grafana-mimir:latest`

## Run

```bash
docker run --rm -p 8080:8080 openvidu/grafana-mimir:latest -config.file=/etc/mimir/local-config.yaml
```
