ARG MIMIR_TAG
FROM grafana/mimir:${MIMIR_TAG}

# Copy shell and grep from busybox (needed for the entrypoint script)
COPY --from=busybox:musl /bin/busybox /bin/sh
COPY --from=busybox:musl /bin/busybox /usr/bin/grep

# Add mimir user (UID 1001) to /etc/passwd
RUN echo 'mimir:x:1001:1001:Mimir:/home/mimir:/sbin/nologin' >> /etc/passwd

WORKDIR /
USER 1001:1001

ENTRYPOINT ["/bin/mimir"]
