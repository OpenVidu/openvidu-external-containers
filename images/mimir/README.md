# openvidu/grafana-mimir

Alpine-based Mimir image that downloads the official `mimir` release binary from Grafana releases.

## Why this image?

OpenVidu needs shell availability for operational scripts and a non-root runtime user.

This image provides:

- Official `mimir` Linux release binary fetched from `grafana/mimir` releases
- SHA-256 verification of the downloaded binary at build time
- Alpine `/bin/sh` support
- A runtime user/group with UID/GID `1001:1001`

## Build

```bash
docker build \
  --build-arg MIMIR_TAG=<tag> \
  -t openvidu/grafana-mimir:<tag> \
  -f images/mimir/Dockerfile \
  .
```

Replace `<tag>` with a valid tag from `grafana/mimir` (for example `3.0.4`).

The build selects the correct binary asset for each platform (`mimir-linux-amd64` / `mimir-linux-arm64`) and verifies it with the corresponding `-sha-256` file.

## Run

```bash
docker run --rm -p 8080:8080 openvidu/grafana-mimir:latest --version
```
