package gcsblob

import (
	"context"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

const (
	defaultEndpoint       = "http://127.0.0.1:4443"
	defaultProjectID      = "goex"
	defaultRequestTimeout = 30 * time.Second
)

type Config struct {
	Endpoint       string
	ProjectID      string
	RequestTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		Endpoint:       envOr("GOEX_GCS_EMULATOR_HOST", envOr("STORAGE_EMULATOR_HOST", defaultEndpoint)),
		ProjectID:      envOr("GOEX_GCS_PROJECT_ID", defaultProjectID),
		RequestTimeout: envDurationOr("GOEX_GCS_REQUEST_TIMEOUT", defaultRequestTimeout),
	}
}

func NewClient(ctx context.Context, cfg Config) (*storage.Client, error) {
	options := []option.ClientOption{option.WithoutAuthentication()}
	if cfg.Endpoint != "" {
		endpoint := strings.TrimRight(cfg.Endpoint, "/") + "/storage/v1/"
		options = append(options, option.WithEndpoint(endpoint))
	}

	return storage.NewClient(ctx, options...)
}

func envOr(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
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
