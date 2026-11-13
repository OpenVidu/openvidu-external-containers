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

## Version updates

Image versions live in `versions.env`. [Renovate](https://docs.renovatebot.com/) keeps them, the base images of the Dockerfiles, the Go modules and the GitHub Actions up to date (configuration in `renovate.json`):

1. Every 3 days the `Renovate` workflow looks for releases that are at least 3 days old.
2. All minor, patch and digest updates go into a single PR, which Renovate merges once CI passes. A major update opens no PR: it waits in the Dependency Dashboard issue, where ticking it opens its PR, to review and merge by hand.
3. Every push to `main` that changes an image runs `Build and push images`, which builds only the images whose contents changed and publishes `<version>`, `<version>-r<N>` and `latest`. `<N>` is the next free build revision on Docker Hub. Updates that do not reach an image publish nothing: Renovate itself, GitHub Actions, the Go modules of the tests, and the Go modules of `images/mc` that the `mc` binary does not link (`.github/scripts/changed-images.sh` decides).
4. Slack gets every image build, published or failed, the PRs Renovate merges, and the failures of Renovate and of CI on `main` or on Renovate's grouped PR. CI failures on major update PRs, which are merged by hand, are not posted.

MongoDB stays on the release series set by `MONGODB_MAJOR_MINOR`. Moving to a new series means changing that value, `MONGODB_VERSION` and the `allowedVersions` rule in `renovate.json`.

Workflows only run on pushes to this repository, never on pull requests from forks.
