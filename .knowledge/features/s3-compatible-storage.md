---
type: feature
title: S3 Compatible Storage
description: YAML and env configuration for AWS S3 Cloudflare R2 MinIO backup targets
resource: storage/s3.go
tags: [plan-b, s3, r2, storage]
timestamp: 2026-07-07T00:00:00Z
---

# S3 Compatible Storage

## Purpose

Backup artifacts upload to **S3-compatible** endpoints: AWS S3, Cloudflare R2, MinIO, etc. Configurable bucket, path prefix, credentials, and local temp path before upload.

## Flows

- **Configure**: `storage.s3` block in `config.yml` or `S3_*` env vars.
- **Upload**: after docker dump completes → `storage/s3.go` puts objects under `backup_prefix`.
- **Download**: restore pulls objects to temp path → arangorestore ingest.

## Main files

- `storage/s3.go`, `storage/interface.go`
- `config/config.go` — parses storage section
- `config.yml.example` — sample R2/S3 endpoint

## Dependencies

- Valid bucket credentials and network egress from runner
- [docker-dump-restore-pipeline](docker-dump-restore-pipeline.md) producing dump directory

## Invariants

- `region: auto` common for R2 — match provider docs.
- Temp path (`path` in config) must have disk space for full dump size.
- Secrets via env override in CI — do not commit live keys (monorepo tracks dev `.env` policy separately).

## Common bugs

- Wrong endpoint URL (missing https) → signature errors.
- Bucket policy denies ListObject → restore cannot find latest backup.
- Path prefix double slashes → objects stored off expected keys.

## Tests

- `config/config_test.go` for config load
- Manual upload to R2 staging bucket

## Related

- [arangodb-backup-restore](arangodb-backup-restore.md)
- Root README environment variables section
