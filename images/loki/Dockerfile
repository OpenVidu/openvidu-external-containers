ARG LOKI_TAG
ARG LOKI_DIGEST

FROM grafana/loki:${LOKI_TAG}${LOKI_DIGEST:+@}${LOKI_DIGEST}

# Copy shell and grep from busybox (needed for the entrypoint script)
COPY --from=busybox:musl@sha256:80e38f9a1b39188b8bdb28879178edb322a4402b081a9aedda1924609decd6f6 /bin/busybox /bin/sh
COPY --from=busybox:musl@sha256:80e38f9a1b39188b8bdb28879178edb322a4402b081a9aedda1924609decd6f6 /bin/busybox /usr/bin/grep

WORKDIR /

USER 10001
