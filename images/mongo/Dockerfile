FROM ubuntu:24.04

ARG YQ_VERSION
ARG WAIT_FOR_PORT_VERSION
ARG RENDER_TEMPLATE_VERSION
ARG MONGODB_MAJOR_MINOR
ARG MONGODB_VERSION
ARG MONGOSH_VERSION

# Define environment variables
ENV YQ_VERSION="${YQ_VERSION}" \
    WAIT_FOR_PORT_VERSION="${WAIT_FOR_PORT_VERSION}" \
    RENDER_TEMPLATE_VERSION="${RENDER_TEMPLATE_VERSION}" \
    MONGODB_MAJOR="${MONGODB_MAJOR_MINOR}" \
    MONGODB_VERSION="${MONGODB_VERSION}" \
    MONGODB_SHELL_VERSION="${MONGOSH_VERSION}" \
    BITNAMI_APP_NAME="mongodb" \
    DEBIAN_FRONTEND=noninteractive

# Copy install script
COPY install.sh /tmp/install.sh

# Run install script
RUN chmod +x /tmp/install.sh && /tmp/install.sh

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
