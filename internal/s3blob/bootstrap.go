package s3blob

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func EnsureBucket(ctx context.Context, client *s3.Client, name string) error {
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &name})
	if err == nil {
		return nil
	}

	var alreadyOwned *types.BucketAlreadyOwnedByYou
	if errors.As(err, &alreadyOwned) {
		return nil
	}

	var alreadyExists *types.BucketAlreadyExists
	if errors.As(err, &alreadyExists) {
		return nil
	}

	return err
}
