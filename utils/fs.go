package utils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/archive"
)

type FS struct{}

func (f *FS) Tar(path string) (io.ReadCloser, error) {
	return archive.TarWithOptions(path, &archive.TarOptions{
		ExcludePatterns: []string{filepath.Join(path, "*.droplet")},
	})
}

func (f *FS) ReadFile(path string) (io.ReadCloser, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	return file, fileInfo.Size(), nil
}

func (f *FS) WriteFile(path string) (io.WriteCloser, error) {
	return os.Create(path)
}
