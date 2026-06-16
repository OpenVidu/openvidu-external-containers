ARG LOKI_TAG

FROM grafana/loki:${LOKI_TAG}

# Copy shell and grep from busybox (needed for the entrypoint script)
COPY --from=busybox:musl /bin/busybox /bin/sh
COPY --from=busybox:musl /bin/busybox /usr/bin/grep

WORKDIR /

USER 10001
