package fs

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
	// TODO: ignore these files in subdirs + obey .cfignore
	return archive.TarWithOptions(absPath, &archive.TarOptions{
		ExcludePatterns: []string{
			"*.droplet",
			".*.cache",
			".cfignore",
			"manifest.yml",
			".gitignore",
			".git",
			".hg",
			".svn",
			"_darcs",
			".DS_Store",
		},
	})
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

func (f *FS) ReadFile(path string) (io.ReadCloser, int64, error) {
	return f.openFile(path, os.O_RDONLY, 0)
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

func (f *FS) WriteFile(path string) (io.WriteCloser, error) {
	return os.Create(path)
}

func (f *FS) MakeDirAll(path string) error {
	return os.MkdirAll(path, 0777)
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
