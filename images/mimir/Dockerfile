# check=skip=InvalidDefaultArgInFrom
# MIMIR_TAG has no default on purpose: the version always comes from
# versions.env, and a default would hide a missing --build-arg.
ARG MIMIR_TAG
ARG MIMIR_DIGEST
FROM grafana/mimir:${MIMIR_TAG}${MIMIR_DIGEST:+@}${MIMIR_DIGEST}

# Copy shell and grep from busybox (needed for the entrypoint script)
COPY --from=busybox:musl@sha256:80e38f9a1b39188b8bdb28879178edb322a4402b081a9aedda1924609decd6f6 /bin/busybox /bin/sh
COPY --from=busybox:musl@sha256:80e38f9a1b39188b8bdb28879178edb322a4402b081a9aedda1924609decd6f6 /bin/busybox /usr/bin/grep

# Add mimir user (UID 1001) to /etc/passwd
RUN echo 'mimir:x:1001:1001:Mimir:/home/mimir:/sbin/nologin' >> /etc/passwd

WORKDIR /
USER 1001:1001

ENTRYPOINT ["/bin/mimir"]
