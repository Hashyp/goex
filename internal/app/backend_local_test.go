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

func TestLocalDeleteRemovesDirectoryRecursively(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "alpha")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	nested := filepath.Join(targetDir, "nested.txt")
	if err := os.WriteFile(nested, []byte("x"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	backend := NewLocalBackend(OSFileSystem{}, root)
	err := backend.Delete(context.Background(), LocalLocation{Path: root}, Entry{
		Name:     "alpha",
		FullPath: "alpha",
		Kind:     KindDirectory,
	})
	if err != nil {
		t.Fatalf("delete returned error: %v", err)
	}

	if _, err := os.Stat(targetDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected directory to be deleted recursively, stat err=%v", err)
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

func TestLocalEnumerateCopyRecursiveDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs")
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write b.txt: %v", err)
	}

	backend := NewLocalBackend(OSFileSystem{}, root)
	entries := []Entry{
		{
			Name:     "docs",
			FullPath: dir,
			Kind:     KindDirectory,
		},
	}
	plan, err := backend.EnumerateCopy(context.Background(), LocalLocation{Path: root}, entries, LocalLocation{Path: filepath.Join(root, "dst")})
	if err != nil {
		t.Fatalf("enumerate copy: %v", err)
	}
	if len(plan) != 2 {
		t.Fatalf("expected 2 files in plan, got %d", len(plan))
	}
}
