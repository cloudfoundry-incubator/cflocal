package fs

import (
	"io"
	"os"
	"path/filepath"
)

type FS struct{}

func (f *FS) ReadFile(path string) (io.ReadCloser, int64, error) {
	return f.openFile(path, os.O_RDONLY, 0)
}

func (f *FS) WriteFile(path string) (io.WriteCloser, error) {
	return os.Create(path)
}

type ReadResetWriteCloser interface {
	io.ReadWriteCloser
	Reset() error
}

func (f *FS) OpenFile(path string) (ReadResetWriteCloser, int64, error) {
	file, size, err := f.openFile(path, os.O_CREATE|os.O_RDWR, 0666)
	return resetFile{file}, size, err
}

type resetFile struct {
	*os.File
}

func (r resetFile) Reset() error {
	if _, err := r.Seek(0, 0); err != nil {
		return err
	}
	return r.Truncate(0)
}

func (f *FS) openFile(path string, flag int, perm os.FileMode) (*os.File, int64, error) {
	file, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, 0, err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	return file, fileInfo.Size(), nil
}

func (f *FS) MakeDirAll(path string) error {
	return os.MkdirAll(path, 0777)
}

func (f *FS) Abs(path string) (string, error) {
	return filepath.Abs(path)
}
