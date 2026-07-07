---
type: feature
title: ArangoDB Backup Restore
description: Cobra backup and restore commands for ArangoDB databases to remote storage
resource: cmd/backup.go
tags: [plan-b, arangodb, backup, restore]
timestamp: 2026-07-07T00:00:00Z
---

# ArangoDB Backup Restore

## Purpose

Primary CLI surface: **`backup`** uploads database dumps; **`restore`** downloads and applies them with safety confirmations. Entry point for ops DR on ArangoDB-backed deployments.

## Flows

- **Backup all**: `arangodb-bk-restore backup` (auto mode, all configured DBs).
- **Backup one**: `backup --database mydb` with optional compress/system flags.
- **Restore**: `restore` with interactive confirms → download from storage → docker restore.
- **Version**: `version` subcommand for support diagnostics.

## Main files

- `cmd/backup.go`, `cmd/restore.go`, `cmd/root.go`, `cmd/version.go`
- `main.go` — cobra entry
- Root `README.md` — usage matrix

## Dependencies

- [docker-dump-restore-pipeline](docker-dump-restore-pipeline.md)
- [s3-compatible-storage](s3-compatible-storage.md)
- [auto-vs-manual-backup-modes](auto-vs-manual-backup-modes.md)

## Invariants

- Restore must not run without explicit confirmations — prevents accidental wipe.
- Backup prefix in config namespaces objects in bucket — keep per-environment prefixes.
- Docker must be available on host running CLI.

## Common bugs

- Wrong `--database` name vs Arango actual → empty dump.
- Restore to production without checking prefix → overwrites live data.
- Missing compress flag parity between backup and restore expectations.

## Tests

- `config/config_test.go`
- Manual backup/restore on staging Arango instance

## Related

- `.knowledge/ARCHITECTURE.md`
- Root `CHANGELOG.md`
