package s3blob

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	defaultEndpoint       = "http://127.0.0.1:9000"
	defaultRegion         = "us-east-1"
	defaultAccessKey      = "minioadmin"
	defaultSecretKey      = "minioadmin"
	defaultPathStyle      = true
	defaultRequestTimeout = 30 * time.Second
)

type Config struct {
	Endpoint       string
	Region         string
	AccessKey      string
	SecretKey      string
	UsePathStyle   bool
	RequestTimeout time.Duration
}

func DefaultConfig() Config {
	cfg := Config{
		Endpoint:       envOr("GOEX_S3_ENDPOINT", defaultEndpoint),
		Region:         envOr("GOEX_S3_REGION", defaultRegion),
		AccessKey:      envOr("GOEX_S3_ACCESS_KEY", defaultAccessKey),
		SecretKey:      envOr("GOEX_S3_SECRET_KEY", defaultSecretKey),
		UsePathStyle:   envBoolOr("GOEX_S3_PATH_STYLE", defaultPathStyle),
		RequestTimeout: envDurationOr("GOEX_S3_REQUEST_TIMEOUT", defaultRequestTimeout),
	}

	if cfg.Endpoint == "" {
		cfg.UsePathStyle = envBoolOr("GOEX_S3_PATH_STYLE", false)
	}

	return cfg
}

func NewClient(ctx context.Context, cfg Config) (*s3.Client, error) {
	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	return client, nil
}

func envOr(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func envBoolOr(name string, fallback bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envDurationOr(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
