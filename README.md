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

1. Every Monday the `Renovate` workflow looks for releases that are at least 3 days old.
2. All minor, patch and digest updates go into a single PR, which Renovate merges once CI passes. Each major update gets its own PR, to review and merge by hand. Open updates are listed in the Dependency Dashboard issue.
3. Every push to `main` that changes an image runs `Build and push images`, which builds only the changed images and publishes `<version>`, `<version>-r<N>` and `latest`. `<N>` is the next free build revision on Docker Hub.
4. Slack gets every image build, published or failed, and the failures of Renovate and of CI on `main` or on Renovate's grouped PR. CI failures on major update PRs, which are merged by hand, are not posted.

MongoDB stays on the release series set by `MONGODB_MAJOR_MINOR`. Moving to a new series means changing that value, `MONGODB_VERSION` and the `allowedVersions` rule in `renovate.json`.

Required secrets:

| Secret | Used by | Purpose |
|---|---|---|
| `RENOVATE_TOKEN` | `Renovate` | Fine-grained personal access token for this repository: read and write access to Contents, Pull requests, Issues, Workflows and Commit statuses, read access to Dependabot alerts. |
| `SLACK_WEBHOOK_URL` | `CI`, `Build and push images`, `Renovate` | Slack incoming webhook for notifications. The optional `SLACK_CHANNEL`, `SLACK_USERNAME` and `SLACK_AVATAR` variables override the channel and the sender. |
| `DOCKERHUB_TOKEN` | `Build and push images` | Docker Hub access token to publish the images, for the user in the `DOCKERHUB_USERNAME` variable. |

Workflows only run on pushes to this repository, never on pull requests from forks.
