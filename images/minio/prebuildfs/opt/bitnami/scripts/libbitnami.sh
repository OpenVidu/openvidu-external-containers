#!/bin/bash
# Copyright Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: APACHE-2.0
#
# Bitnami custom library

# shellcheck disable=SC1091

# Load Generic Libraries
. /opt/bitnami/scripts/liblog.sh

# Constants
BOLD='\033[1m'

# Functions

########################
# Print the welcome page
# Globals:
#   DISABLE_WELCOME_MESSAGE
#   BITNAMI_APP_NAME
# Arguments:
#   None
# Returns:
#   None
#########################
print_welcome_page() {
    if [[ -z "${DISABLE_WELCOME_MESSAGE:-}" ]]; then
        if [[ -n "$BITNAMI_APP_NAME" ]]; then
            print_image_welcome_page
        fi
    fi
}

########################
# Print the welcome page for a Bitnami Docker image
# Globals:
#   BITNAMI_APP_NAME
# Arguments:
#   None
# Returns:
#   None
#########################
print_image_welcome_page() {
    info ""
    info "${BOLD}OpenVidu MinIO container — powered by the Chainguard MinIO fork${RESET}"
    info ""
    info "  This image ships custom binaries built from source:"
    info "    · MinIO server — https://github.com/chainguard-forks/minio"
    info "    · mc client    — a minimal implementation containing only the mc commands"
    info "                     required for this image to operate"
    info ""
    info "  It is fully compatible with the official Bitnami MinIO container."
    info ""
    info "  Source: https://github.com/OpenVidu/openvidu-external-containers"
    info ""
}

