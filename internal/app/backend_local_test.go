package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDeleteRemovesFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "alpha.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	backend := NewLocalBackend(OSFileSystem{}, root)
	err := backend.Delete(context.Background(), LocalLocation{Path: root}, Entry{
		Name:     "alpha.txt",
		FullPath: "alpha.txt",
		Kind:     KindObject,
	})
	if err != nil {
		t.Fatalf("delete returned error: %v", err)
	}

	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be deleted, stat err=%v", err)
	}
}

func TestLocalDeleteReturnsInvalidLocation(t *testing.T) {
	backend := NewLocalBackend(OSFileSystem{}, t.TempDir())
	err := backend.Delete(context.Background(), AzureLocation{Mode: AzureModeContainers}, Entry{
		Name: "alpha.txt",
		Kind: KindObject,
	})
	if !errors.Is(err, ErrInvalidLocation) {
		t.Fatalf("expected ErrInvalidLocation, got %v", err)
	}
}
