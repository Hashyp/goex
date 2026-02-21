package main

import (
	"io/fs"
	"os"
)

type FileSystem interface {
	ReadDir(name string) ([]os.DirEntry, error)
	Stat(name string) (fs.FileInfo, error)
}

type OSFileSystem struct{}

func (OSFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func (OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}
