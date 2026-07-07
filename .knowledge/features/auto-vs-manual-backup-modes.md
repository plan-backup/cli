---
type: feature
title: Auto vs Manual Backup Modes
description: general.mode controls all-databases vs single-database backup targeting
resource: config/config.go
tags: [plan-b, config, backup, modes]
timestamp: 2026-07-07T00:00:00Z
---

# Auto vs Manual Backup Modes

## Purpose

`general.mode` in config chooses whether **`backup`** iterates all databases listed in YAML or expects explicit `--database` targeting (manual workflows).

## Flows

- **Auto**: `mode: auto` + database list → backup command dumps each entry sequentially.
- **Manual**: `mode: manual` → operator passes `--database` for one DB; `default_database` fallback.
- **Cron**: auto mode suited for scheduled jobs; manual for ad-hoc single-DB exports.

## Main files

- `config/config.go`, `config/config_test.go`
- `config.yml.example` — `general.mode`, `default_database`, database array
- `cmd/backup.go` — interprets mode + flags

## Dependencies

- [arangodb-backup-restore](arangodb-backup-restore.md)
- ArangoDB credentials in `database.arangodb` section

## Invariants

- Auto mode requires non-empty database list — empty list backs up nothing silently if misconfigured.
- `_system` inclusion controlled by `--include-system` flag defaults.
- Mode in config does not change restore confirmations — restore always interactive.

## Common bugs

- Auto mode but only one DB intended → forgotten `--database` in manual mode dumps all listed DBs.
- `_system` omitted when needed for full cluster restore.
- Wrong `default_database` in manual scripts.

## Tests

- `config/config_test.go`

## Related

- [docker-dump-restore-pipeline](docker-dump-restore-pipeline.md)
- [s3-compatible-storage](s3-compatible-storage.md)
