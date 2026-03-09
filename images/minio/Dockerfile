FROM golang:1.24-alpine AS builder

ARG MINIO_TAG

WORKDIR /src

RUN apk add --no-cache git && \
    git clone --depth 1 --branch "${MINIO_TAG}" https://github.com/chainguard-forks/minio.git .

RUN COMMIT=$(git rev-parse HEAD) && \
    SHORT_COMMIT=$(git rev-parse --short HEAD) && \
    CGO_ENABLED=0 go build -tags kqueue -trimpath \
      -ldflags "-s -w \
        -X github.com/chainguard-forks/minio/cmd.Version=${MINIO_TAG} \
        -X github.com/chainguard-forks/minio/cmd.ReleaseTag=${MINIO_TAG} \
        -X github.com/chainguard-forks/minio/cmd.CommitID=${COMMIT} \
        -X github.com/chainguard-forks/minio/cmd.ShortCommitID=${SHORT_COMMIT} \
        -X github.com/chainguard-forks/minio/cmd.Edition=agpl" \
      -o /minio .

FROM ubuntu:24.04

ENV YQ_VERSION=v4.52.4 \
    WAIT_FOR_PORT_VERSION=v1.0.10 \
    RENDER_TEMPLATE_VERSION=v1.0.9 \
    DEBIAN_FRONTEND=noninteractive

# COPY --from=builder /minio /usr/local/bin/minio

RUN mkdir /data && chown minio:minio /data

VOLUME /data

EXPOSE 9000 9001

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q --spider http://localhost:9000/minio/health/live || exit 1

USER minio

ENTRYPOINT ["/usr/local/bin/minio"]
CMD ["server", "/data"]
