package azureblob

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

func EnsureContainer(ctx context.Context, client *azblob.Client, name string) error {
	_, err := client.CreateContainer(ctx, name, nil)
	if err == nil {
		return nil
	}

	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) {
		if strings.EqualFold(responseErr.ErrorCode, "ContainerAlreadyExists") || strings.EqualFold(responseErr.ErrorCode, "ContainerBeingDeleted") {
			return nil
		}
	}

	if strings.Contains(strings.ToLower(err.Error()), "containeralreadyexists") {
		return nil
	}

	return err
}
