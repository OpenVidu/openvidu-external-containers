# openvidu/grafana-loki

`grafana/loki` image with `/bin/sh` for OpenVidu deployment scripts.

## Why this image?

The official `grafana/loki` image is distroless — it has no shell. OpenVidu's Loki entrypoint script uses `/bin/sh` for conditional checks (`grep`, shell operators) before starting the loki process.

This image provides:
- Official `loki` binary from `grafana/loki`
- `/bin/sh` (busybox, statically linked)
- `grep` (busybox, statically linked)
- Same `10001` UID as the official image

## Build

```bash
docker build \
  --build-arg LOKI_TAG=<VERSION> \
  -t openvidu/grafana-loki:<VERSION> \
  -f images/loki/Dockerfile \
  .
```

Where `<VERSION>` is the desired Loki version.

In CI/release workflows, tags are sourced from `versions.env` and published as:

- `openvidu/grafana-loki:${LOKI_TAG}`
- `openvidu/grafana-loki:${LOKI_TAG}-r${LOKI_BUILD_NUMBER}`
- `openvidu/grafana-loki:latest`

## Run

```bash
docker run --rm -p 3100:3100 openvidu/grafana-loki:latest --config.file=/etc/loki/local-config.yaml
```
