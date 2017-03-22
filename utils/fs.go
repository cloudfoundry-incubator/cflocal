package utils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/archive"
)

type FS struct{}

func (f *FS) Tar(path string) (io.ReadCloser, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return archive.TarWithOptions(absPath, &archive.TarOptions{
		ExcludePatterns: []string{filepath.Join(path, "*.droplet")},
		ChownOpts:       &archive.TarChownOptions{UID: 2000, GID: 2000},
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

func (f *FS) MakeDirAll(path string) error {
	return os.MkdirAll(path, 0750)
}

func (f *FS) IsDirEmpty(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	if _, err := file.Readdirnames(1); err != io.EOF {
		return false, err
	}
	return true, nil
}

func (f *FS) Abs(path string) (string, error) {
	return filepath.Abs(path)
}
