FROM golang:1.27-alpine AS minio-builder

ARG MINIO_TAG

WORKDIR /src

RUN apk add --no-cache git && \
    git clone --depth 1 --branch "${MINIO_TAG}" https://github.com/chainguard-forks/minio.git .

# The fork pins github.com/apache/thrift with a replace directive to a commit
# before v0.24.0, which carries the security fix; v0.24.0 contains that commit.
# Empty it once a MinIO release replaces thrift with v0.24.0 or later.
ARG MINIO_THRIFT_REPLACE="github.com/apache/thrift@v0.24.0"
RUN if [ -n "${MINIO_THRIFT_REPLACE}" ]; then \
      go mod edit -replace "github.com/apache/thrift=${MINIO_THRIFT_REPLACE}" && \
      go mod download github.com/apache/thrift; \
    fi

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

FROM golang:1.27-alpine AS mc-builder

WORKDIR /src

COPY images/mc/go.mod images/mc/go.sum ./
RUN go mod download
COPY images/mc/*.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mc .

FROM chainguard/wolfi-base:latest@sha256:ae9575b121f5776e2f04e192064bc2c5665af9d1c425d620dc773aca31698712

ENV HOME="/" \
    OS_ARCH="amd64" \
    OS_FLAVOUR="wolfi" \
    OS_NAME="linux"

COPY images/minio/prebuildfs /

RUN apk add --no-cache \
      bash \
      ca-certificates-bundle \
      curl \
      jq \
      procps \
      shadow

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
