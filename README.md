# ArangoDB Backup & Restore Tool

[![CI](https://github.com/apito-io/arangodb-bk-restore/actions/workflows/ci.yml/badge.svg)](https://github.com/apito-io/arangodb-bk-restore/actions/workflows/ci.yml)
[![Release](https://github.com/apito-io/arangodb-bk-restore/actions/workflows/release.yml/badge.svg)](https://github.com/apito-io/arangodb-bk-restore/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/apito-io/arangodb-bk-restore)](https://goreportcard.com/report/github.com/apito-io/arangodb-bk-restore)
[![codecov](https://codecov.io/gh/apito-io/arangodb-bk-restore/branch/main/graph/badge.svg)](https://codecov.io/gh/apito-io/arangodb-bk-restore)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A powerful CLI tool for backing up and restoring ArangoDB databases with S3-compatible storage support (AWS S3, Cloudflare R2, MinIO, etc.).

## ✨ Features

- **🗄️ ArangoDB Support**: Backup and restore ArangoDB databases using Docker-based `arangodump` and `arangorestore`
- **☁️ S3-Compatible Storage**: Support for AWS S3, Cloudflare R2, MinIO, and other S3-compatible storage services
- **🔒 Secure**: Multiple confirmation prompts for restore operations to prevent accidental data loss
- **⚙️ Configurable**: YAML-based configuration with environment variable support
- **🔄 Auto & Manual Modes**: Automated backups for multiple databases or manual single-database operations
- **📦 Cross-Platform**: Binaries available for Linux, macOS, and Windows
- **🐳 Docker Support**: Available as a Docker container
- **📊 Detailed Logging**: Comprehensive logging with progress tracking

## 🚀 Installation

### Using Go Install

```bash
go install github.com/apito-io/arangodb-bk-restore@latest
```

### Using Homebrew (macOS/Linux)

```bash
brew tap apito-io/tools
brew install arangodb-bk-restore
```

### Using Docker

```bash
docker pull ghcr.io/apito-io/arangodb-bk-restore:latest
```

### Download Binary

Download the latest binary from the [releases page](https://github.com/apito-io/arangodb-bk-restore/releases).

## 📋 Prerequisites

- **Docker**: Required for running ArangoDB dump/restore operations
- **ArangoDB**: Target ArangoDB instance (3.x or later)
- **S3-Compatible Storage**: AWS S3, Cloudflare R2, MinIO, etc.

## ⚙️ Configuration

Create a `config.yml` file in your working directory:

```yaml
general:
  mode: "manual" # auto or manual
  default_database: "arangodb"
  default_storage: "s3"
  backup_prefix: "your-prefix/arangodb"

database:
  arangodb:
    host: localhost
    port: 8529
    username: root
    password: your-arangodb-password
    database:
      - "_system"
      - "your_database_1"
      - "your_database_2"

storage:
  s3:
    endpoint: https://your-s3-endpoint.com
    bucket: your-backup-bucket
    access_key: your-access-key
    secret_key: your-secret-key
    region: auto
    path: /tmp/backup-temp
```

### Environment Variables

You can override configuration values using environment variables:

```bash
export ARANGODB_HOST=localhost
export ARANGODB_PORT=8529
export ARANGODB_USERNAME=root
export ARANGODB_PASSWORD=password
export S3_ENDPOINT=https://your-endpoint.com
export S3_BUCKET=your-bucket
export S3_ACCESS_KEY=your-access-key
export S3_SECRET_KEY=your-secret-key
```

## 🔧 Usage

### Backup Operations

#### Backup All Databases (Auto Mode)

```bash
arangodb-bk-restore backup
```

#### Backup Specific Database

```bash
arangodb-bk-restore backup --database my_database
```

#### Backup with Custom Options

```bash
arangodb-bk-restore backup \
  --database my_database \
  --include-system \
  --compress \
  --output-dir /custom/path
```

### Restore Operations

#### Interactive Restore (Recommended)

```bash
arangodb-bk-restore restore
```

This will:
1. List available backups
2. Allow you to select a backup
3. Show target database connection details
4. Require multiple confirmations
5. Download, extract, and restore the backup

#### Restore Specific Backup

```bash
arangodb-bk-restore restore --backup-key "prefix/database_20240101_120000.tar.gz"
```

#### Restore to Different Database

```bash
arangodb-bk-restore restore \
  --backup-key "prefix/database_20240101_120000.tar.gz" \
  --database new_database_name
```

## 🐳 Docker Usage

### Using Docker Compose

```yaml
version: '3.8'

services:
  backup:
    image: ghcr.io/apito-io/arangodb-bk-restore:latest
    volumes:
      - ./config.yml:/config.yml
      - ./backups:/tmp/backup-temp
    command: backup
    environment:
      - ARANGODB_HOST=arangodb
      - ARANGODB_PASSWORD=your-password
    depends_on:
      - arangodb

  arangodb:
    image: arangodb:3.11.8
    environment:
      - ARANGO_ROOT_PASSWORD=your-password
    ports:
      - "8529:8529"
```

### Direct Docker Run

```bash
# Backup
docker run --rm \
  -v $(pwd)/config.yml:/config.yml \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/apito-io/arangodb-bk-restore:latest backup

# Restore
docker run -it --rm \
  -v $(pwd)/config.yml:/config.yml \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/apito-io/arangodb-bk-restore:latest restore
```

## 📖 Command Reference

### Global Flags

- `--config string`: Config file path (default: ./config.yml)
- `--verbose, -v`: Enable verbose output

### Backup Command

```bash
arangodb-bk-restore backup [flags]
```

**Flags:**
- `--database, -d string`: Specific database to backup
- `--compress, -c`: Compress backup (default: true)
- `--include-system, -s`: Include system collections (default: true)
- `--overwrite, -o`: Overwrite existing backups
- `--output-dir string`: Output directory for backups

### Restore Command

```bash
arangodb-bk-restore restore [flags]
```

**Flags:**
- `--backup-key, -k string`: Specific backup key to restore
- `--database, -d string`: Target database name for restore
- `--include-system, -s`: Include system collections (default: true)
- `--input-dir, -i string`: Input directory for restore
- `--overwrite, -o`: Overwrite existing database

## 🔐 Security Best Practices

1. **Use Environment Variables**: Store sensitive credentials in environment variables
2. **Restrict S3 Permissions**: Use IAM policies to limit S3 access to backup bucket only
3. **Enable Encryption**: Use S3 bucket encryption and ArangoDB TLS
4. **Regular Testing**: Regularly test restore procedures
5. **Access Logging**: Enable S3 access logging for audit trails

## 🛠️ Development

### Prerequisites

- Go 1.21 or later
- Docker
- Make (optional)

### Building from Source

```bash
git clone https://github.com/apito-io/arangodb-bk-restore.git
cd arangodb-bk-restore
go mod download
go build -o arangodb-bk-restore .
```

### Running Tests

```bash
go test -v ./...
```

### Using Make

```bash
# Build
make build

# Run tests
make test

# Build for all platforms
make build-all

# Clean
make clean
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/apito-io/arangodb-bk-restore/issues)
- **Discussions**: [GitHub Discussions](https://github.com/apito-io/arangodb-bk-restore/discussions)
- **Email**: support@apito.io

## 🗺️ Roadmap

- [ ] PostgreSQL support
- [ ] MySQL support
- [ ] Web UI for backup management
- [ ] Scheduled backups with cron-like syntax
- [ ] Backup encryption
- [ ] Incremental backups
- [ ] Backup verification and integrity checks
- [ ] Multi-cloud storage support
- [ ] Slack/Discord notifications

## 📊 Changelog

See [CHANGELOG.md](CHANGELOG.md) for a list of changes and releases.

---

Made with ❤️ by [Apito](https://apito.io)