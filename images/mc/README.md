# mc

A minimal, OpenVidu-focused drop-in replacement for the official [`minio/mc`](https://min.io/docs/minio/linux/reference/minio-mc.html) client.

## Why

The upstream `minio/mc` image has become unmaintained. This image ships only the commands actually used by OpenVidu deployments, built from the official [minio-go](https://github.com/minio/minio-go) and [madmin-go](https://github.com/minio/madmin-go) SDKs.

## Supported commands

This is a **very minimal implementation**. Only the commands required by OpenVidu deployments are supported.

### `alias`

```
mc alias set   <name> <url> <access-key> <secret-key>
mc alias ls
mc alias remove <name>
```

Manages server aliases persisted in `$HOME/.mc/config.json` (override with `--config-dir`).
The config format is compatible with the upstream `mc` client.

### `ls`

```
mc ls <alias>
mc ls <alias>/<bucket>
mc ls <alias>/<bucket>/<prefix>
```

Lists buckets (when no bucket given) or objects. Exits 0 if the bucket exists (even if empty), non-zero otherwise. Used by OpenVidu to wait until a bucket is available.

### `stat`

```
mc stat <alias>/<bucket>
mc stat <alias>/<bucket>/<object>
```

Exits 0 if the bucket or object exists, non-zero otherwise.

### `mirror`

```
mc mirror [--overwrite] [--max-workers <n>] <source> <target>
```

Syncs objects between a MinIO bucket and a local directory. Exactly one side must be a MinIO alias target; the other must be a local path. `--max-workers` controls the number of concurrent transfers (default: number of CPUs).

```bash
# Backup: MinIO bucket → local directory
mc mirror --overwrite openvidu/mybucket /backup/mybucket

# Restore: local directory → MinIO bucket
mc mirror --overwrite /backup/mybucket openvidu/mybucket
```

Without `--overwrite`, files/objects that already exist at the destination are skipped. Used by OpenVidu backup and restore scripts.

### `rb`

```
mc rb [--force] [--dangerous] <alias>/<bucket> [<alias>/<bucket> ...]
```

Removes one or more buckets. Accepts multiple targets in a single invocation.

Without `--force` the bucket must be empty; otherwise the command exits non-zero.
With `--force` all objects (and their versions) are deleted before the bucket is removed.
`--dangerous` is accepted for CLI compatibility with the upstream client.

```bash
# Remove an empty bucket
mc rb openvidu/mybucket

# Remove a non-empty bucket and all its contents
mc rb --force openvidu/mybucket
```

### `mb`

```
mc mb [--region <region>] [--ignore-existing | -p] <alias>/<bucket>
```

Creates a bucket, optionally in a specific region. With `--ignore-existing` (or `-p`) the command exits 0 silently when the bucket already exists.

### `anonymous`

```
mc anonymous set <policy> <alias>/<bucket>/
mc anonymous get <alias>/<bucket>/
```

Sets or gets the anonymous access policy for a bucket. Allowed policies: `none` (or `private`), `download`, `upload`, `public`.

### `admin info`

```
mc admin info <alias> [--json]
```

Prints server information. Human-readable output includes uptime, version, network/drive health, a per-pool usage table, and a usage summary. With `--json` the output is a JSON object with `"status": "success"`. Used by OpenVidu to wait until MinIO is ready.

### `admin service`

```
mc admin service restart <alias>
mc admin service stop    <alias>
```

Restarts or stops the MinIO server.

## Global flags

| Flag / Env var | Description |
|---|---|
| `--config-dir <path>` | Directory for `config.json` (default: `$HOME/.mc`) |
| `MC_CONFIG_DIR` | Env var equivalent of `--config-dir`; flag takes precedence |
| `--quiet` / `-q` | Suppress non-error output |

## Not supported

Everything else — `cp`, `policy`, `admin user`, `admin group`, `watch`, etc. If you need those, use the official `minio/mc` client.
