package app

import (
	"context"
	"fmt"
	"os"

	"defaultdevcontainer/internal/azureblob"
	"defaultdevcontainer/internal/gcsblob"
	"defaultdevcontainer/internal/s3blob"
)

type paneBackendChoice int

const (
	paneBackendFilesystem paneBackendChoice = iota
	paneBackendAzure
	paneBackendS3
	paneBackendGCS
)

var paneBackendChoices = []paneBackendChoice{
	paneBackendFilesystem,
	paneBackendAzure,
	paneBackendS3,
	paneBackendGCS,
}

func paneBackendLabel(choice paneBackendChoice) string {
	switch choice {
	case paneBackendFilesystem:
		return "file system"
	case paneBackendAzure:
		return "azure"
	case paneBackendS3:
		return "s3"
	case paneBackendGCS:
		return "gcs"
	default:
		return "unknown"
	}
}

func paneBackendForChoice(choice paneBackendChoice, localStartPath string) PaneBackend {
	switch choice {
	case paneBackendFilesystem:
		return NewLocalBackend(OSFileSystem{}, localStartPath)
	case paneBackendAzure:
		client, err := azureblob.NewClient()
		if err != nil {
			return NewStaticErrorBackendWithLocation(
				fmt.Errorf("failed to initialize azure client: %w", err),
				AzureLocation{Mode: AzureModeContainers},
				"azure:/",
			)
		}
		return NewAzureBlobBackend(client)
	case paneBackendS3:
		s3Config := s3blob.DefaultConfig()
		client, err := s3blob.NewClient(context.Background(), s3Config)
		if err != nil {
			return NewStaticErrorBackendWithLocation(
				fmt.Errorf("failed to initialize s3 client: %w", err),
				S3Location{Mode: S3ModeBuckets},
				"s3:///",
			)
		}
		return NewS3Backend(client, s3Config.RequestTimeout)
	case paneBackendGCS:
		gcsConfig := gcsblob.DefaultConfig()
		client, err := gcsblob.NewClient(context.Background(), gcsConfig)
		if err != nil {
			return NewStaticErrorBackendWithLocation(
				fmt.Errorf("failed to initialize gcs client: %w", err),
				GCSLocation{Mode: GCSModeBuckets},
				"gcs:///",
			)
		}
		return NewGCSBackend(client, gcsConfig.ProjectID, gcsConfig.RequestTimeout)
	default:
		return NewStaticErrorBackend(fmt.Errorf("unsupported backend selection"))
	}
}

func paneBackendChoiceFromPane(p Pane) paneBackendChoice {
	switch p.backend.(type) {
	case LocalBackend:
		return paneBackendFilesystem
	case AzureBlobBackend:
		return paneBackendAzure
	case S3Backend:
		return paneBackendS3
	case GCSBackend:
		return paneBackendGCS
	case StaticErrorBackend:
		switch p.location.(type) {
		case LocalLocation:
			return paneBackendFilesystem
		case AzureLocation:
			return paneBackendAzure
		case S3Location:
			return paneBackendS3
		case GCSLocation:
			return paneBackendGCS
		default:
			return paneBackendFilesystem
		}
	default:
		return paneBackendFilesystem
	}
}

func currentWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	return cwd
}

func paneName(id activePane) string {
	if id == paneRight {
		return "right"
	}

	return "left"
}

func findPaneBackendChoiceIndex(choice paneBackendChoice) int {
	for index, candidate := range paneBackendChoices {
		if candidate == choice {
			return index
		}
	}

	return 0
}
