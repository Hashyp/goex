package gcsblob

import (
	"context"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
)

func EnsureBucket(ctx context.Context, client *storage.Client, projectID string, name string) error {
	err := client.Bucket(name).Create(ctx, projectID, nil)
	if err == nil {
		return nil
	}

	if apiErr, ok := err.(*googleapi.Error); ok {
		if apiErr.Code == 409 {
			return nil
		}
	}

	if strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return nil
	}

	return err
}
