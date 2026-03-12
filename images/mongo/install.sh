#!/bin/bash

set -e

# Determine architecture
ARCH=$(dpkg --print-architecture)
UBUNTU_CODENAME=$(grep VERSION_CODENAME /etc/os-release | cut -d'=' -f2)

# Install dependencies
apt-get update && apt-get install -y wget gnupg curl

# Create directory for MongoDB binaries
mkdir -p /opt/bitnami/mongodb/bin

# Install MongoDB
curl -fsSL https://www.mongodb.org/static/pgp/server-"${MONGODB_MAJOR}".asc | \
    gpg --dearmor -o /usr/share/keyrings/mongodb-server-"${MONGODB_MAJOR}".gpg
echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-${MONGODB_MAJOR}.gpg ] https://repo.mongodb.org/apt/ubuntu ${UBUNTU_CODENAME}/mongodb-org/${MONGODB_MAJOR} multiverse" | \
    tee /etc/apt/sources.list.d/mongodb-org-"${MONGODB_MAJOR}".list
apt-get update
apt-get install -y mongodb-org-server="${MONGODB_VERSION}"
ln -sf /usr/bin/mongod /opt/bitnami/mongodb/bin/mongod

# Install MongoDB Shell
ARCH2=$( [ "$ARCH" = "amd64" ] && echo "x64" || echo "arm64" )
wget -q https://downloads.mongodb.com/compass/mongosh-"${MONGODB_SHELL_VERSION}"-linux-"${ARCH2}".tgz
tar xvzf mongosh-"${MONGODB_SHELL_VERSION}"-linux-"${ARCH2}".tgz > /dev/null
mv mongosh-"${MONGODB_SHELL_VERSION}"-linux-"${ARCH2}"/bin/* /opt/bitnami/mongodb/bin/

# Clean up
apt-get remove -y wget curl gnupg
apt-get autoremove -y
apt-get clean
rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* mongosh-"${MONGODB_SHELL_VERSION}"-linux-"${ARCH2}".tgz
