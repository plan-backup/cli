package storage

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"arangodb-bk-restore/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"
)

// S3Storage implements the StorageInterface for S3-compatible storage
type S3Storage struct {
	config config.S3Config
	client *s3.Client
	logger *logrus.Logger
}

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(cfg config.S3Config, logger *logrus.Logger) (*S3Storage, error) {
	// For Cloudflare R2, we need to handle the region differently
	region := cfg.Region
	if region == "auto" {
		region = "us-east-1" // R2 default region
	}

	// Create AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     region,
				HostnameImmutable: true,
				PartitionID:       "aws",
			}, nil
		})),
		awsconfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     cfg.AccessKey,
				SecretAccessKey: cfg.SecretKey,
			},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %v", err)
	}

	// Create S3 client with R2-specific options
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = false // R2 uses virtual-hosted style
		o.Region = region
	})

	return &S3Storage{
		config: cfg,
		client: client,
		logger: logger,
	}, nil
}

// Upload uploads a backup to S3 storage
func (s *S3Storage) Upload(ctx context.Context, localPath, remoteKey string) error {
	s.logger.Infof("Uploading backup from %s to %s", localPath, remoteKey)

	// Check if local path exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("local path does not exist: %s", localPath)
	}

	// Create tar.gz archive if localPath is a directory
	archivePath, err := s.createArchive(localPath)
	if err != nil {
		return fmt.Errorf("failed to create archive: %v", err)
	}
	defer os.Remove(archivePath) // Clean up temporary archive

	// Open file for reading
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Get file info for content length
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Upload to S3
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.config.Bucket),
		Key:           aws.String(remoteKey),
		Body:          file,
		ContentLength: aws.Int64(fileInfo.Size()),
		ContentType:   aws.String("application/gzip"),
		Metadata: map[string]string{
			"original-path": localPath,
			"upload-time":   time.Now().Format(time.RFC3339),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %v", err)
	}

	s.logger.Infof("Successfully uploaded backup to %s", remoteKey)
	return nil
}

// Download downloads a backup from S3 storage
func (s *S3Storage) Download(ctx context.Context, remoteKey, localPath string) error {
	s.logger.Infof("Downloading backup from %s to %s", remoteKey, localPath)

	// Create local directory if it doesn't exist
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %v", err)
	}

	// Download from S3
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(remoteKey),
	})
	if err != nil {
		return fmt.Errorf("failed to download from S3: %v", err)
	}
	defer result.Body.Close()

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to copy content: %v", err)
	}

	s.logger.Infof("Successfully downloaded backup to %s", localPath)
	return nil
}

// ListBackups lists available backups with optional prefix filtering
func (s *S3Storage) ListBackups(ctx context.Context, prefix string) ([]*BackupMetadata, error) {
	s.logger.Infof("Listing backups with prefix: %s", prefix)

	var backups []*BackupMetadata
	var continuationToken *string

	for {
		// List objects
		input := &s3.ListObjectsV2Input{
			Bucket:  aws.String(s.config.Bucket),
			Prefix:  aws.String(prefix),
			MaxKeys: aws.Int32(1000),
		}

		if continuationToken != nil {
			input.ContinuationToken = continuationToken
		}

		result, err := s.client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %v", err)
		}

		// Process objects
		for _, obj := range result.Contents {
			if strings.HasSuffix(*obj.Key, ".tar.gz") {
				metadata, err := s.parseBackupKey(*obj.Key)
				if err != nil {
					s.logger.Warnf("Failed to parse backup key %s: %v", *obj.Key, err)
					continue
				}

				metadata.Key = *obj.Key
				metadata.Size = *obj.Size
				metadata.LastModified = *obj.LastModified

				backups = append(backups, metadata)
			}
		}

		// Check if there are more objects
		if result.IsTruncated == nil || !*result.IsTruncated {
			break
		}
		continuationToken = result.NextContinuationToken
	}

	s.logger.Infof("Found %d backups", len(backups))
	return backups, nil
}

// DeleteBackup deletes a backup from storage
func (s *S3Storage) DeleteBackup(ctx context.Context, remoteKey string) error {
	s.logger.Infof("Deleting backup: %s", remoteKey)

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(remoteKey),
	})

	if err != nil {
		return fmt.Errorf("failed to delete backup: %v", err)
	}

	s.logger.Infof("Successfully deleted backup: %s", remoteKey)
	return nil
}

// GetBackupInfo returns information about a backup
func (s *S3Storage) GetBackupInfo(ctx context.Context, remoteKey string) (*BackupMetadata, error) {
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(remoteKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %v", err)
	}

	metadata, err := s.parseBackupKey(remoteKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse backup key: %v", err)
	}

	metadata.Key = remoteKey
	metadata.Size = *result.ContentLength
	metadata.LastModified = *result.LastModified

	return metadata, nil
}

// TestConnection tests the storage connection
func (s *S3Storage) TestConnection(ctx context.Context) error {
	s.logger.Info("Testing S3 connection")

	// Try to list objects (limited to 1)
	_, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.config.Bucket),
		MaxKeys: aws.Int32(1),
	})

	if err != nil {
		// Provide more specific error information for R2
		if strings.Contains(err.Error(), "NoSuchBucket") {
			return fmt.Errorf("S3 connection test failed: bucket '%s' not found. Please check:\n1. Bucket name is correct\n2. Bucket exists in your R2 account\n3. Access key has permissions to access this bucket", s.config.Bucket)
		}
		if strings.Contains(err.Error(), "AccessDenied") {
			return fmt.Errorf("S3 connection test failed: access denied. Please check:\n1. Access key and secret key are correct\n2. Access key has permissions to list objects in bucket '%s'", s.config.Bucket)
		}
		return fmt.Errorf("S3 connection test failed: %v", err)
	}

	s.logger.Info("S3 connection test successful")
	return nil
}

// createArchive creates a tar.gz archive from a directory
func (s *S3Storage) createArchive(sourcePath string) (string, error) {
	// Create temporary archive file
	archivePath := sourcePath + ".tar.gz"
	file, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to create archive file: %v", err)
	}
	defer file.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through source directory
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create header
		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return fmt.Errorf("failed to create tar header: %v", err)
		}

		// Update header name to be relative to source
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %v", err)
		}

		// Write file content if it's a regular file
		if !info.IsDir() {
			sourceFile, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open source file: %v", err)
			}
			defer sourceFile.Close()

			if _, err := io.Copy(tarWriter, sourceFile); err != nil {
				return fmt.Errorf("failed to copy file content: %v", err)
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to create archive: %v", err)
	}

	return archivePath, nil
}

// ExtractArchive extracts a tar.gz file to a target directory
func (s *S3Storage) ExtractArchive(archivePath, targetDir string) error {
	s.logger.Infof("Extracting archive %s to %s", archivePath, targetDir)

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Open archive file
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %v", err)
	}
	defer file.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		// Skip entries that are just "." or ".."
		if header.Name == "." || header.Name == ".." {
			continue
		}

		// Create target path
		targetPath := filepath.Join(targetDir, header.Name)

		// Ensure the target path is within the target directory (security check)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(targetDir)+string(os.PathSeparator)) && filepath.Clean(targetPath) != filepath.Clean(targetDir) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
		case tar.TypeReg:
			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %v", err)
			}

			// Create file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %v", err)
			}

			// Copy content
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to copy file content: %v", err)
			}
			outFile.Close()

			// Set file permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set file permissions: %v", err)
			}
		}
	}

	s.logger.Infof("Successfully extracted archive to %s", targetDir)
	return nil
}

// parseBackupKey parses backup key to extract metadata
// Expected format: prefix/engine/database_name_YYYYMMDD_HHMMSS.tar.gz
// or legacy format: prefix/database_name_YYYYMMDD_HHMMSS.tar.gz
func (s *S3Storage) parseBackupKey(key string) (*BackupMetadata, error) {
	// Expected format: prefix/engine/database_timestamp.tar.gz
	parts := strings.Split(key, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid backup key format: %s", key)
	}

	filename := parts[len(parts)-1]
	if !strings.HasSuffix(filename, ".tar.gz") {
		return nil, fmt.Errorf("invalid backup file extension: %s", filename)
	}

	// Remove .tar.gz extension
	baseName := strings.TrimSuffix(filename, ".tar.gz")

	// Use regex to extract database name and timestamp more reliably
	// Pattern: database_name_YYYYMMDD_HHMMSS
	// Example: apito_prod_projects_20250821_130436

	// Find the last occurrence of _YYYYMMDD_HHMMSS pattern
	re := regexp.MustCompile(`_(\d{8}_\d{6})$`)
	matches := re.FindStringSubmatch(baseName)

	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid backup filename format, expected _YYYYMMDD_HHMMSS pattern: %s", baseName)
	}

	timestampStr := matches[1] // 20250821_130436

	// Database name is everything before the timestamp pattern
	databaseName := strings.TrimSuffix(baseName, "_"+timestampStr)

	// Parse timestamp
	timestamp, err := time.Parse("20060102_150405", timestampStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp '%s': %v", timestampStr, err)
	}

	metadata := BackupMetadata{
		DatabaseName: databaseName,
		Timestamp:    timestamp,
	}

	return &metadata, nil
}

