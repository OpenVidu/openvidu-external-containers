ARG MIMIR_TAG

FROM busybox:musl AS shell

ARG MIMIR_TAG

FROM grafana/mimir:${MIMIR_TAG}
COPY --from=shell /bin/busybox /bin/sh
COPY --from=shell /bin/busybox /usr/bin/grep
COPY --from=shell /bin/busybox /sbin/nologin
COPY --from=shell /bin/busybox /usr/bin/id
COPY --from=shell /bin/busybox /usr/bin/groups
COPY --from=shell /bin/busybox /usr/bin/head

# Add mimir user (UID 1001) to /etc/passwd
RUN echo 'mimir:x:1001:1001:Mimir:/home/mimir:/sbin/nologin' >> /etc/passwd

WORKDIR /
USER 1001:1001

ENTRYPOINT ["/bin/mimir"]
