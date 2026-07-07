# plan-b-cli — Architecture

Distilled from root `README.md` and Go module `arangodb-bk-restore`.

## Overview

**plan-b-cli** (`arangodb-bk-restore`) backs up and restores **ArangoDB** databases using Docker-hosted `arangodump` / `arangorestore`, uploading artifacts to **S3-compatible** storage (AWS S3, Cloudflare R2, MinIO). Supports **auto** (all configured DBs) and **manual** (single DB) modes via YAML + env overrides.

## Pipeline

```
config.yml → database/arangodb.go (docker dump/restore)
          → storage/s3.go (upload/download)
          → cmd/backup.go | cmd/restore.go
```

## Key components

- **`config/config.go`** — YAML schema: `general.mode`, database list, S3 credentials.
- **`database/arangodb.go`** — Docker wrapper around arangodump/arangorestore.
- **`storage/s3.go`** — S3-compatible client (endpoint, bucket, keys, region).
- **`cmd/root.go`** — cobra root; backup/restore subcommands with flags.
- **`config.yml.example`** — documented defaults and env var mapping.

## Modes

| Mode | Behavior |
|------|----------|
| `auto` | Backup all databases listed under `database.arangodb.database` |
| `manual` | Target one DB via `--database` flag |

Restore uses multiple confirmation prompts to prevent accidental overwrite.

## Distribution

Go install, Homebrew tap `apito-io/tools`, Docker image `ghcr.io/apito-io/arangodb-bk-restore`, GoReleaser cross-platform binaries.

## Consumers

- Apito deployments still on ArangoDB legacy stacks
- Ops runbooks for disaster recovery outside engine's primary SQL/libsql path

Last Updated: 2026-07-07
