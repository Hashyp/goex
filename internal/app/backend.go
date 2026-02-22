package app

import "context"
import "time"

type PaneBackend interface {
	List(ctx context.Context, state Location, showHidden bool) ([]Entry, error)
	Enter(ctx context.Context, state Location, highlighted Entry) (Location, bool, error)
	Parent(state Location) (Location, bool)
	ParentHighlightName(state Location) string
	DisplayPath(state Location) string
	LoadTimeout() time.Duration
	InitialLocation() Location
}
