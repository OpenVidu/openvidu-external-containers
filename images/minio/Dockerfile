FROM golang:1.24-alpine AS minio-builder

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

FROM golang:1.26-alpine AS mc-builder

WORKDIR /src

COPY images/mc/go.mod images/mc/go.sum ./
RUN go mod download
COPY images/mc/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mc .

FROM debian:bookworm-slim

ENV HOME="/" \
    OS_ARCH="amd64" \
    OS_FLAVOUR="debian-12" \
    OS_NAME="linux" \
    DEBIAN_FRONTEND=noninteractive

COPY images/minio/prebuildfs /

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      jq \
      procps \
    && rm -rf /var/lib/apt/lists/*

COPY images/minio/rootfs /

RUN mkdir -p /opt/bitnami/minio/bin /opt/bitnami/minio-client/bin

COPY --from=minio-builder /minio /opt/bitnami/minio/bin/minio
COPY --from=mc-builder /mc /opt/bitnami/minio-client/bin/mc

RUN chmod +x /opt/bitnami/minio/bin/minio /opt/bitnami/minio-client/bin/mc

RUN useradd -r -u 1001 -g root minio

RUN /opt/bitnami/scripts/minio-client/postunpack.sh && \
    /opt/bitnami/scripts/minio/postunpack.sh

ENV BITNAMI_APP_NAME="minio" \
    PATH="/opt/bitnami/minio-client/bin:/opt/bitnami/minio/bin:${PATH}"

EXPOSE 9000 9001

VOLUME ["/bitnami/minio/data"]

USER 1001

ENTRYPOINT ["/opt/bitnami/scripts/minio/entrypoint.sh"]
CMD ["/opt/bitnami/scripts/minio/run.sh"]
