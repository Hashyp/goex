package app

import (
	"context"
	"fmt"
	"time"
)

type StaticErrorBackend struct {
	err         error
	location    Location
	displayPath string
}

func NewStaticErrorBackend(err error) StaticErrorBackend {
	return NewStaticErrorBackendWithLocation(err, AzureLocation{Mode: AzureModeContainers}, "azure:/")
}

func NewStaticErrorBackendWithLocation(err error, location Location, displayPath string) StaticErrorBackend {
	return StaticErrorBackend{
		err:         err,
		location:    location,
		displayPath: displayPath,
	}
}

func (b StaticErrorBackend) InitialLocation() Location {
	if b.location == nil {
		return AzureLocation{Mode: AzureModeContainers}
	}

	return b.location
}

func (b StaticErrorBackend) List(_ context.Context, _ Location, _ bool) ([]Entry, error) {
	if b.err == nil {
		return nil, fmt.Errorf("backend unavailable")
	}
	return nil, b.err
}

func (b StaticErrorBackend) Enter(_ context.Context, state Location, _ Entry) (Location, bool, error) {
	return state, false, nil
}

func (b StaticErrorBackend) Delete(_ context.Context, _ Location, _ Entry) error {
	if b.err == nil {
		return fmt.Errorf("backend unavailable")
	}

	return b.err
}

func (b StaticErrorBackend) Parent(state Location) (Location, bool) {
	return state, false
}

func (b StaticErrorBackend) ParentHighlightName(_ Location) string {
	return ""
}

func (b StaticErrorBackend) DisplayPath(_ Location) string {
	if b.displayPath == "" {
		return "backend:/"
	}

	return b.displayPath
}

func (b StaticErrorBackend) LoadTimeout() time.Duration {
	return 10 * time.Second
}
