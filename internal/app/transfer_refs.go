package app

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

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
		return TransferObjectRef{
			Provider: "azure",
			Scope:    loc.Container,
			Path:     objectPath,
			Display:  "azure:/" + loc.Container + "/" + objectPath,
		}, nil
	case S3Location:
		if loc.Mode != S3ModeObjects || loc.Bucket == "" {
			return TransferObjectRef{}, fmt.Errorf("s3 destination bucket not selected")
		}
		objectPath := joinObjectPath(loc.Prefix, normalizedRelative)
		return TransferObjectRef{
			Provider: "s3",
			Scope:    loc.Bucket,
			Path:     objectPath,
			Display:  "s3:///" + loc.Bucket + "/" + objectPath,
		}, nil
	case GCSLocation:
		if loc.Mode != GCSModeObjects || loc.Bucket == "" {
			return TransferObjectRef{}, fmt.Errorf("gcs destination bucket not selected")
		}
		objectPath := joinObjectPath(loc.Prefix, normalizedRelative)
		return TransferObjectRef{
			Provider: "gcs",
			Scope:    loc.Bucket,
			Path:     objectPath,
			Display:  "gcs:///" + loc.Bucket + "/" + objectPath,
		}, nil
	default:
		return TransferObjectRef{}, fmt.Errorf("unsupported destination location type: %T", destination)
	}
}

func sourceRefForLocation(state Location, objectPath string) (TransferObjectRef, error) {
	switch loc := state.(type) {
	case LocalLocation:
		return TransferObjectRef{
			Provider: "local",
			Scope:    loc.Path,
			Path:     objectPath,
			Display:  objectPath,
		}, nil
	case AzureLocation:
		if loc.Container == "" {
			return TransferObjectRef{}, fmt.Errorf("azure source container not selected")
		}
		return TransferObjectRef{
			Provider: "azure",
			Scope:    loc.Container,
			Path:     objectPath,
			Display:  "azure:/" + loc.Container + "/" + objectPath,
		}, nil
	case S3Location:
		if loc.Bucket == "" {
			return TransferObjectRef{}, fmt.Errorf("s3 source bucket not selected")
		}
		return TransferObjectRef{
			Provider: "s3",
			Scope:    loc.Bucket,
			Path:     objectPath,
			Display:  "s3:///" + loc.Bucket + "/" + objectPath,
		}, nil
	case GCSLocation:
		if loc.Bucket == "" {
			return TransferObjectRef{}, fmt.Errorf("gcs source bucket not selected")
		}
		return TransferObjectRef{
			Provider: "gcs",
			Scope:    loc.Bucket,
			Path:     objectPath,
			Display:  "gcs:///" + loc.Bucket + "/" + objectPath,
		}, nil
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
