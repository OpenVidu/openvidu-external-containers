ARG LOKI_TAG

FROM grafana/loki:${LOKI_TAG} AS loki
FROM busybox:musl AS shell

FROM grafana/loki:${LOKI_TAG}

COPY --from=loki /usr/bin/loki /usr/bin/loki
COPY --from=shell /bin/busybox /bin/sh
COPY --from=shell /bin/busybox /usr/bin/grep

WORKDIR /

USER 10001
