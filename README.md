# openvidu-external-containers

Purpose-built container images used by OpenVidu deployments.

## Images

| Image | Description |
|---|---|
| [`mc`](images/mc/README.md) | Minimal drop-in replacement for `minio/mc` with only the commands needed by OpenVidu deployments |
| [`mimir`](images/mimir/README.md) | Alpine-based image with official `mimir` release binaries, SHA-256 verification, and non-root execution (UID/GID `1001:1001`) |
| [`mongo`](images/mongo/README.md) | Ubuntu-based Bitnami-compatible MongoDB image for OpenVidu deployments |
| [`minio`](images/minio/README.md) | MinIO server built from the chainguard-forks/minio fork at a specified tag |

## Security Scan Report

*Last updated: 2026-03-12 20:28 UTC*

### Summary

| Image | Critical | High | Medium | Low | Total |
|-------|----------|------|--------|-----|-------|
| mc | 0 | 0 | 0 | 0 | 0 |
| mimir | 0 | 0 | 0 | 0 | 0 |
| minio | 0 | 0 | 0 | 0 | 0 |
| mongo | 0 | 0 | 8 | 6 | 14 |
| **TOTAL** | **0** | **0** | **8** | **6** | **14** |

---
### Detailed Vulnerabilities by Image

<details>
<summary><strong>mc</strong> (0 vulnerabilities)</summary>

No vulnerabilities found.

</details>

<details>
<summary><strong>mimir</strong> (0 vulnerabilities)</summary>

No vulnerabilities found.

</details>

<details>
<summary><strong>minio</strong> (0 vulnerabilities)</summary>

No vulnerabilities found.

</details>

<details>
<summary><strong>mongo</strong> (14 vulnerabilities)</summary>

| Severity | CVE ID | Package | Installed | Fixed | Title |
|----------|--------|---------|-----------|-------|-------|
| MEDIUM | CVE-2025-68972 | gpgv | 2.4.4-2ubuntu17.4 | — | gnupg: GnuPG: Signature bypass via form feed character in signed messages |
| MEDIUM | CVE-2025-14831 | libgnutls30t64 | 3.8.3-1.1ubuntu3.4 | 3.8.3-1.1ubuntu3.5 | gnutls: GnuTLS: Denial of Service via excessive resource consumption during cert... |
| MEDIUM | CVE-2025-8941 | libpam-modules | 1.5.3-5ubuntu5.5 | — | linux-pam: Incomplete fix for CVE-2025-6020 |
| MEDIUM | CVE-2025-8941 | libpam-modules-bin | 1.5.3-5ubuntu5.5 | — | linux-pam: Incomplete fix for CVE-2025-6020 |
| MEDIUM | CVE-2025-8941 | libpam-runtime | 1.5.3-5ubuntu5.5 | — | linux-pam: Incomplete fix for CVE-2025-6020 |
| MEDIUM | CVE-2025-8941 | libpam0g | 1.5.3-5ubuntu5.5 | — | linux-pam: Incomplete fix for CVE-2025-6020 |
| MEDIUM | CVE-2026-3731 | libssh-4 | 0.10.6-2ubuntu0.3 | — | libssh: libssh: Denial of Service via out-of-bounds read in SFTP extension name ... |
| MEDIUM | CVE-2025-45582 | tar | 1.35+dfsg-3build1 | — | tar: Tar path traversal |
| LOW | CVE-2016-2781 | coreutils | 9.4-3ubuntu6.1 | — | coreutils: Non-privileged session can escape to the parent session in chroot |
| LOW | CVE-2022-3219 | gpgv | 2.4.4-2ubuntu17.4 | — | gnupg: denial of service issue (resource consumption) using compressed packets |
| LOW | CVE-2024-2236 | libgcrypt20 | 1.10.3-2build1 | — | libgcrypt: vulnerable to Marvin Attack |
| LOW | CVE-2025-9820 | libgnutls30t64 | 3.8.3-1.1ubuntu3.4 | 3.8.3-1.1ubuntu3.5 | gnutls: Stack-based Buffer Overflow in gnutls_pkcs11_token_init() Function |
| LOW | CVE-2024-56433 | login | 1:4.13+dfsg1-4ubuntu3.2 | — | shadow-utils: Default subordinate ID configuration in /etc/login.defs could lead... |
| LOW | CVE-2024-56433 | passwd | 1:4.13+dfsg1-4ubuntu3.2 | — | shadow-utils: Default subordinate ID configuration in /etc/login.defs could lead... |

</details>

*Vulnerability scanning is informational only.*
