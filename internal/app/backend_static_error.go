package app

import (
	"context"
	"fmt"
)

type StaticErrorBackend struct {
	err error
}

func NewStaticErrorBackend(err error) StaticErrorBackend {
	return StaticErrorBackend{err: err}
}

func (b StaticErrorBackend) InitialLocation() Location {
	return AzureLocation{Mode: AzureModeContainers}
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

func (b StaticErrorBackend) Parent(state Location) (Location, bool) {
	return state, false
}

func (b StaticErrorBackend) DisplayPath(_ Location) string {
	return "azure:/"
}
