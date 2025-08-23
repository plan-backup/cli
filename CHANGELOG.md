# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of ArangoDB Backup & Restore Tool
- Support for ArangoDB backup and restore operations using Docker
- S3-compatible storage support (AWS S3, Cloudflare R2, MinIO)
- Interactive restore mode with backup selection
- Multiple confirmation prompts for restore operations
- YAML configuration with environment variable overrides
- Auto and manual backup modes
- Cross-platform binaries (Linux, macOS, Windows)
- Docker container support
- Comprehensive logging and progress tracking
- CLI built with Cobra framework
- GoReleaser configuration for automated releases
- GitHub Actions workflows for CI/CD

### Features
- **Backup Operations**:
  - Single database backup
  - Multiple database backup (auto mode)
  - System collections inclusion option
  - Compression support
  - Custom output directory support

- **Restore Operations**:
  - Interactive backup selection
  - Target database connection information display
  - Multiple confirmation prompts for safety
  - Automatic archive extraction
  - Custom target database naming
  - System collections restoration

- **Storage Support**:
  - S3-compatible storage (AWS S3, Cloudflare R2, MinIO)
  - Automatic tar.gz compression
  - Secure upload/download with progress tracking
  - Backup metadata parsing and validation

- **Configuration**:
  - YAML-based configuration
  - Environment variable overrides
  - Multiple database and storage configurations
  - Flexible backup prefix settings

- **Security**:
  - Multiple confirmation prompts for destructive operations
  - Masked password display
  - Secure credential handling
  - Path traversal protection in archive extraction

### Technical Details
- Built with Go 1.22
- Uses AWS SDK v2 for S3 operations
- Docker-based ArangoDB operations
- Structured logging with Logrus
- Cross-platform build support
- Comprehensive error handling

## [1.0.0] - 2024-XX-XX (Planned)

### Added
- Initial stable release
