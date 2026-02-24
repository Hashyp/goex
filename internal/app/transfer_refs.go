package app

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

func newTransferObjectRef(provider, scope, objectPath string) (TransferObjectRef, error) {
	display, err := transferObjectDisplay(provider, scope, objectPath)
	if err != nil {
		return TransferObjectRef{}, err
	}

	return TransferObjectRef{
		Provider: provider,
		Scope:    scope,
		Path:     objectPath,
		Display:  display,
	}, nil
}

func transferObjectDisplay(provider, scope, objectPath string) (string, error) {
	switch provider {
	case "local":
		return objectPath, nil
	case "azure":
		return "azure:/" + scope + "/" + objectPath, nil
	case "s3":
		return "s3:///" + scope + "/" + objectPath, nil
	case "gcs":
		return "gcs:///" + scope + "/" + objectPath, nil
	default:
		return "", fmt.Errorf("unsupported transfer provider: %q", provider)
	}
}

func resolveDestinationRef(destination Location, relative string) (TransferObjectRef, error) {
	normalizedRelative := normalizeTransferRelative(relative)
	switch loc := destination.(type) {
	case LocalLocation:
		if loc.Path == "" {
			return TransferObjectRef{}, fmt.Errorf("local destination path is empty")
		}
		fullPath := filepath.Join(loc.Path, filepath.FromSlash(normalizedRelative))
		return TransferObjectRef{
			Provider: "local",
			Scope:    loc.Path,
			Path:     fullPath,
			Display:  fullPath,
		}, nil
	case AzureLocation:
		if loc.Mode != AzureModeObjects || loc.Container == "" {
			return TransferObjectRef{}, fmt.Errorf("azure destination container not selected")
		}
		objectPath := joinObjectPath(loc.Prefix, normalizedRelative)
		return newTransferObjectRef("azure", loc.Container, objectPath)
	case S3Location:
		if loc.Mode != S3ModeObjects || loc.Bucket == "" {
			return TransferObjectRef{}, fmt.Errorf("s3 destination bucket not selected")
		}
		objectPath := joinObjectPath(loc.Prefix, normalizedRelative)
		return newTransferObjectRef("s3", loc.Bucket, objectPath)
	case GCSLocation:
		if loc.Mode != GCSModeObjects || loc.Bucket == "" {
			return TransferObjectRef{}, fmt.Errorf("gcs destination bucket not selected")
		}
		objectPath := joinObjectPath(loc.Prefix, normalizedRelative)
		return newTransferObjectRef("gcs", loc.Bucket, objectPath)
	default:
		return TransferObjectRef{}, fmt.Errorf("unsupported destination location type: %T", destination)
	}
}

func sourceRefForLocation(state Location, objectPath string) (TransferObjectRef, error) {
	switch loc := state.(type) {
	case LocalLocation:
		return newTransferObjectRef("local", loc.Path, objectPath)
	case AzureLocation:
		if loc.Container == "" {
			return TransferObjectRef{}, fmt.Errorf("azure source container not selected")
		}
		return newTransferObjectRef("azure", loc.Container, objectPath)
	case S3Location:
		if loc.Bucket == "" {
			return TransferObjectRef{}, fmt.Errorf("s3 source bucket not selected")
		}
		return newTransferObjectRef("s3", loc.Bucket, objectPath)
	case GCSLocation:
		if loc.Bucket == "" {
			return TransferObjectRef{}, fmt.Errorf("gcs source bucket not selected")
		}
		return newTransferObjectRef("gcs", loc.Bucket, objectPath)
	default:
		return TransferObjectRef{}, fmt.Errorf("unsupported source location type: %T", state)
	}
}

func normalizeTransferRelative(relative string) string {
	cleaned := strings.ReplaceAll(relative, "\\", "/")
	cleaned = strings.TrimPrefix(cleaned, "/")
	cleaned = path.Clean(cleaned)
	if cleaned == "." {
		return ""
	}

	return cleaned
}

func joinObjectPath(prefix, relative string) string {
	rel := normalizeTransferRelative(relative)
	base := strings.TrimPrefix(strings.ReplaceAll(prefix, "\\", "/"), "/")
	if base == "" {
		return rel
	}
	if rel == "" {
		return strings.TrimSuffix(base, "/")
	}

	return path.Join(base, rel)
}
