---
type: feature
title: Docker Dump Restore Pipeline
description: Docker-wrapped arangodump and arangorestore execution against configured host
resource: database/arangodb.go
tags: [plan-b, docker, arangodb, dump]
timestamp: 2026-07-07T00:00:00Z
---

# Docker Dump Restore Pipeline

## Purpose

Consistent **arangodump/arangorestore** versions via Docker containers — avoids host-installed Arango tools mismatch. Connects to configured host/port with credentials from config or env.

## Flows

- **Dump**: spawn container → run arangodump → write to mounted temp dir → upload to S3.
- **Restore**: download backup → mount into container → arangorestore into target DB.
- **Flags**: compress, include-system, overwrite map to dump/restore CLI args.

## Main files

- `database/arangodb.go`, `database/interface.go`
- `cmd/backup.go`, `cmd/restore.go` — orchestration + logging (logrus)
- `Dockerfile` — optional containerized CLI runner

## Dependencies

- Docker daemon on execution host
- Network reachability to ArangoDB `host:port`
- [s3-compatible-storage](s3-compatible-storage.md) for artifact transfer

## Invariants

- ArangoDB 3.x+ supported per README prerequisites.
- Container must mount same path for dump output and S3 upload reader.
- Restore confirmations happen **before** container restore starts.

## Common bugs

- Docker not in PATH in cron → silent failure — wrap with explicit docker check.
- Arango auth failure inside container → empty dump directory uploaded anyway if not validated.
- Platform image pull blocked in air-gapped env — pre-pull images in ops runbook.

## Tests

- Manual dump against local Arango docker compose
- CI workflow builds binary; integration optional

## Related

- [arangodb-backup-restore](arangodb-backup-restore.md)
- [auto-vs-manual-backup-modes](auto-vs-manual-backup-modes.md)
