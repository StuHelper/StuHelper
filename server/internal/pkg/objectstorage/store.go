package objectstorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Config 对象存储配置。
type Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	ForcePathStyle  bool
	PresignTTL      time.Duration
}

// Store S3 兼容对象存储客户端。
type Store struct {
	client     *s3.Client
	presign    *s3.PresignClient
	bucket     string
	presignTTL time.Duration
}

// New 创建对象存储客户端。
func New(ctx context.Context, cfg Config) (*Store, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, errors.New("object storage endpoint is required")
	}
	if strings.TrimSpace(cfg.Region) == "" {
		return nil, errors.New("object storage region is required")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("object storage bucket is required")
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return nil, errors.New("object storage credentials are required")
	}
	if cfg.PresignTTL <= 0 {
		cfg.PresignTTL = 10 * time.Minute
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if !strings.Contains(endpoint, "://") {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		endpoint = scheme + "://" + endpoint
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load object storage config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = &endpoint
		o.UsePathStyle = cfg.ForcePathStyle
	})

	return &Store{
		client:     client,
		presign:    s3.NewPresignClient(client),
		bucket:     cfg.Bucket,
		presignTTL: cfg.PresignTTL,
	}, nil
}

// EnsureBucket 确保存储桶存在。
func (s *Store) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	if err == nil {
		return nil
	}

	var apiErr *types.NotFound
	if !errors.As(err, &apiErr) {
		lower := strings.ToLower(err.Error())
		if !strings.Contains(lower, "not found") && !strings.Contains(lower, "no such bucket") {
			return fmt.Errorf("head bucket %q: %w", s.bucket, err)
		}
	}

	_, createErr := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &s.bucket,
	})
	if createErr != nil && !strings.Contains(strings.ToLower(createErr.Error()), "bucket already owned") {
		return fmt.Errorf("create bucket %q: %w", s.bucket, createErr)
	}
	return nil
}

// Upload 上传对象。
func (s *Store) Upload(ctx context.Context, key string, content []byte, contentType string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("object key is required")
	}
	if len(content) == 0 {
		return errors.New("object content is empty")
	}
	size := int64(len(content))
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          bytes.NewReader(content),
		ContentLength: &size,
		ContentType:   &contentType,
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

// PresignGetURL 生成下载签名链接。
func (s *Store) PresignGetURL(ctx context.Context, key string) (string, error) {
	result, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}, func(o *s3.PresignOptions) {
		o.Expires = s.presignTTL
	})
	if err != nil {
		return "", fmt.Errorf("presign get %q: %w", key, err)
	}
	return result.URL, nil
}
