ARG MIMIR_TAG=3.0.4

FROM alpine:3.21 AS mimir-prepeare

ARG MIMIR_TAG
ARG TARGETARCH

RUN apk add --no-cache ca-certificates curl

RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64|arm64) ;; \
      *) echo "Unsupported TARGETARCH: ${TARGETARCH}"; exit 1 ;; \
    esac; \
    release_base_url="https://github.com/grafana/mimir/releases/download/mimir-${MIMIR_TAG}"; \
    curl -fsSL "${release_base_url}/mimir-linux-${TARGETARCH}" -o /tmp/mimir; \
    curl -fsSL "${release_base_url}/mimir-linux-${TARGETARCH}-sha-256" -o /tmp/mimir.sha256; \
    echo "$(cat /tmp/mimir.sha256)  /tmp/mimir" | sha256sum -c -; \
    chmod +x /tmp/mimir

FROM alpine:3.21

RUN apk add --no-cache ca-certificates && \
    addgroup -S -g 1001 mimir && \
    adduser -S -D -H -u 1001 -G mimir -s /bin/sh mimir

COPY --from=mimir-prepeare /tmp/mimir /bin/mimir

EXPOSE 8080

USER 1001:1001

ENTRYPOINT ["/bin/mimir"]
