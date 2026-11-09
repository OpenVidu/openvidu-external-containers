FROM golang:1.27-alpine AS yq-builder

ARG YQ_VERSION

WORKDIR /src

RUN apk add --no-cache git && \
    git clone https://github.com/mikefarah/yq.git . && \
    git checkout "refs/tags/${YQ_VERSION}" && \
    test "$(git rev-parse HEAD)" = "$(git rev-list -n 1 "refs/tags/${YQ_VERSION}")"

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /yq .

FROM golang:1.27-alpine AS wait-for-port-builder

ARG WAIT_FOR_PORT_COMMIT

WORKDIR /src

RUN apk add --no-cache git && \
    git clone https://github.com/bitnami/wait-for-port.git . && \
    git checkout "${WAIT_FOR_PORT_COMMIT}" && \
    test "$(git rev-parse HEAD)" = "${WAIT_FOR_PORT_COMMIT}"

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /wait-for-port .

FROM golang:1.27-alpine AS render-template-builder

ARG RENDER_TEMPLATE_COMMIT

WORKDIR /src

RUN apk add --no-cache git && \
    git clone https://github.com/bitnami/render-template.git . && \
    git checkout "${RENDER_TEMPLATE_COMMIT}" && \
    test "$(git rev-parse HEAD)" = "${RENDER_TEMPLATE_COMMIT}"

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /render-template .

FROM ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852 AS mongodb

ARG MONGODB_MAJOR_MINOR
ARG MONGODB_VERSION
ARG MONGOSH_VERSION

# Define environment variables
ENV MONGODB_MAJOR="${MONGODB_MAJOR_MINOR}" \
    MONGODB_VERSION="${MONGODB_VERSION}" \
    MONGODB_SHELL_VERSION="${MONGOSH_VERSION}" \
    BITNAMI_APP_NAME="mongodb" \
    DEBIAN_FRONTEND=noninteractive

# Copy install script
COPY install.sh /tmp/install.sh

# Run install script
RUN chmod +x /tmp/install.sh && /tmp/install.sh

COPY --from=yq-builder /yq /usr/local/bin/yq
COPY --from=wait-for-port-builder /wait-for-port /usr/local/bin/wait-for-port
COPY --from=render-template-builder /render-template /usr/local/bin/render-template
RUN chmod +x /usr/local/bin/yq /usr/local/bin/wait-for-port /usr/local/bin/render-template

# Set up Bitnami scripts and permissions
COPY prebuildfs /
RUN chmod g+rwX /opt/bitnami && find / -perm /6000 -type f -exec chmod a-s {} \; || true
COPY rootfs /
RUN /opt/bitnami/scripts/mongodb/postunpack.sh

# Expose MongoDB port
EXPOSE 27017

# Set user and commands
USER 1001
ENTRYPOINT [ "/opt/bitnami/scripts/mongodb/entrypoint.sh" ]
CMD [ "/opt/bitnami/scripts/mongodb/run.sh" ]

# -compat variant (--target compat): the same MongoDB, run the way the official
# mongo image runs it, for deployments written for that image. The entrypoint
# starts as root, hands /data/db and /data/configdb to the mongodb user (UID and
# GID 999, as in the official image) and drops to it with setpriv.
FROM mongodb AS compat
USER root
RUN groupmod -g 999 mongodb && \
    usermod -u 999 -g 999 -d /data/db mongodb && \
    mkdir -p /data/db /data/configdb /docker-entrypoint-initdb.d && \
    chown mongodb:mongodb /data/db /data/configdb && \
    ln -s /opt/bitnami/mongodb/bin/mongosh /usr/local/bin/mongosh
COPY compat/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENV HOME=/data/db
VOLUME [ "/data/db", "/data/configdb" ]
EXPOSE 27017
ENTRYPOINT [ "docker-entrypoint.sh" ]
CMD [ "mongod" ]

# The default target stays the Bitnami-style image above
FROM mongodb
