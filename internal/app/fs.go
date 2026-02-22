package app

import (
	"io/fs"
	"os"
)

type FileSystem interface {
	ReadDir(name string) ([]os.DirEntry, error)
	Stat(name string) (fs.FileInfo, error)
	Remove(name string) error
	RemoveAll(path string) error
}

type OSFileSystem struct{}

func (OSFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func (OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (OSFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}
