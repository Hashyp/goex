package app

import "context"
import "io"
import "time"

type PaneBackend interface {
	List(ctx context.Context, state Location, showHidden bool) ([]Entry, error)
	Enter(ctx context.Context, state Location, highlighted Entry) (Location, bool, error)
	Delete(ctx context.Context, state Location, highlighted Entry) error
	Parent(state Location) (Location, bool)
	ParentHighlightName(state Location) string
	DisplayPath(state Location) string
	LoadTimeout() time.Duration
	InitialLocation() Location
}

// CopyEnumerator expands selected entries into copy plan items.
// Implementations may recurse through directories/prefixes.
type CopyEnumerator interface {
	EnumerateCopy(ctx context.Context, state Location, selected []Entry, destination Location) ([]TransferPlanItem, error)
}

// CopyReader provides read streams for copy sources.
type CopyReader interface {
	OpenCopyReader(ctx context.Context, source TransferObjectRef) (CopyReadHandle, error)
}

// CopyWriter provides write streams and conflict checks for copy destinations.
type CopyWriter interface {
	CopyDestinationExists(ctx context.Context, destination TransferObjectRef) (bool, error)
	OpenCopyWriter(ctx context.Context, destination TransferObjectRef, metadata TransferObjectMetadata) (io.WriteCloser, error)
}
