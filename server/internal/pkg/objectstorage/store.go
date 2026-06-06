package objectstorage

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	TLSCAFile       string
}

// Store S3 兼容对象存储客户端。
type Store struct {
	client     *s3.Client
	presign    *s3.PresignClient
	bucket     string
	presignTTL time.Duration
}

type ObjectInfo struct {
	Key         string
	SizeBytes   int64
	ContentType string
}

// New 创建对象存储客户端。
func New(ctx context.Context, cfg Config) (*Store, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, &StoreError{Kind: ErrorKindConfig, Op: "init", Resource: "endpoint", Err: errors.New("object storage endpoint is required")}
	}
	if strings.TrimSpace(cfg.Region) == "" {
		return nil, &StoreError{Kind: ErrorKindConfig, Op: "init", Resource: "region", Err: errors.New("object storage region is required")}
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, &StoreError{Kind: ErrorKindConfig, Op: "init", Resource: "bucket", Err: errors.New("object storage bucket is required")}
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return nil, &StoreError{Kind: ErrorKindConfig, Op: "init", Resource: "credentials", Err: errors.New("object storage credentials are required")}
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

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	}
	if strings.TrimSpace(cfg.TLSCAFile) != "" {
		caBundle, err := readRootCABundle(strings.TrimSpace(cfg.TLSCAFile))
		if err != nil {
			return nil, wrapError("load_ca_bundle", cfg.TLSCAFile, err)
		}
		loadOptions = append(loadOptions, awsconfig.WithCustomCABundle(bytes.NewReader(caBundle)))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, wrapError("load_config", cfg.Endpoint, err)
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

func readRootCABundle(caFile string) ([]byte, error) {
	// #nosec G304 -- caFile is an operator-supplied deployment configuration path.
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read ca bundle: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("ca bundle does not contain PEM certificates")
	}
	return caCert, nil
}

// CheckBucket 验证存储桶存在且可访问，不会自动创建。
// 应用启动时使用此方法做 fail-closed 校验；bucket 由 infra/deploy 流程预置。
func (s *Store) CheckBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	if err != nil {
		return wrapError("check_bucket", s.bucket, err)
	}
	return nil
}

// Upload 上传对象。
func (s *Store) Upload(ctx context.Context, key string, content []byte, contentType string) error {
	if strings.TrimSpace(key) == "" {
		return &StoreError{Kind: ErrorKindConfig, Op: "upload", Resource: "object", Err: errors.New("object key is required")}
	}
	if len(content) == 0 {
		return &StoreError{Kind: ErrorKindConfig, Op: "upload", Resource: key, Err: errors.New("object content is empty")}
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
		return wrapError("put_object", key, err)
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
		return "", wrapError("presign_get", key, err)
	}
	return result.URL, nil
}

func (s *Store) Stat(ctx context.Context, key string) (*ObjectInfo, error) {
	output, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, wrapError("head_object", key, err)
	}
	info := &ObjectInfo{Key: key}
	if output.ContentLength != nil {
		info.SizeBytes = *output.ContentLength
	}
	if output.ContentType != nil {
		info.ContentType = *output.ContentType
	}
	return info, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return wrapError("delete_object", key, err)
	}
	return nil
}
