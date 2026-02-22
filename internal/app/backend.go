package app

import "context"

type PaneBackend interface {
	List(ctx context.Context, state Location, showHidden bool) ([]Entry, error)
	Enter(ctx context.Context, state Location, highlighted Entry) (Location, bool, error)
	Parent(state Location) (Location, bool)
	DisplayPath(state Location) string
	InitialLocation() Location
}
