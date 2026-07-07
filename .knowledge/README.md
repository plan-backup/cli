# plan-b-cli — Knowledge

Part of the **apito** ecosystem. See `/.knowledge/projects/apito.md` for how this repo fits and its blast radius.

## Read order
1. This file. 2. `ARCHITECTURE.md`. 3. `DECISIONS.md`. 4. `features/README.md`. 5. `../.memory/CURRENT.md` and `../.memory/HANDOFF.md`.

## Purpose

CLI tool **`arangodb-bk-restore`** for ArangoDB backup/restore to S3-compatible storage — Docker-based dump/restore, auto/manual modes, YAML + env configuration. See root `README.md` for install and usage examples.

## Responsibilities

- Wrap `arangodump` / `arangorestore` in Docker for consistent tooling
- Upload/download backup artifacts via S3-compatible APIs (R2, MinIO, AWS)
- Guard restore with confirmation prompts; support compress and system collections flags

## Consumers / blast radius

| Consumer | Impact |
|----------|--------|
| Legacy ArangoDB Apito stacks | DR and migration backups |
| CI/release | GoReleaser binaries and Docker image tags |

## Reasoning archive

- Historical Cursor plans distilled into this knowledge base live in `archive/plans/`.

Last Updated: 2026-07-07
