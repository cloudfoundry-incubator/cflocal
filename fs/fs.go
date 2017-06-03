package fs

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"code.cloudfoundry.org/cli/cf/appfiles"
	"github.com/docker/docker/pkg/archive"
)

type FS struct{}

func (f *FS) TarApp(path string) (app io.ReadCloser, err error) {
	var absPath, appDir string

	absPath, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	zipper := appfiles.ApplicationZipper{}
	if zipper.IsZipFile(absPath) {
		appDir, err = ioutil.TempDir("", "cflocal-zip")
		if err != nil {
			return nil, err
		}
		defer func() {
			if err != nil {
				os.RemoveAll(appDir)
				return
			}
			app = &closeWrapper{
				ReadCloser: app,
				After: func() error {
					return os.RemoveAll(appDir)
				},
			}
		}()
		if err := zipper.Unzip(absPath, appDir); err != nil {
			return nil, err
		}
	} else {
		appDir, err = filepath.EvalSymlinks(absPath)
		if err != nil {
			return nil, err
		}
	}
	files, err := appFiles(appDir)
	if err != nil {
		return nil, err
	}
	return archive.TarWithOptions(appDir, &archive.TarOptions{
		IncludeFiles: files,
	})
}

type closeWrapper struct {
	io.ReadCloser
	After func() error
}

func (c *closeWrapper) Close() (err error) {
	defer func() {
		if afterErr := c.After(); err == nil {
			err = afterErr
		}
	}()
	return c.ReadCloser.Close()
}

func appFiles(path string) ([]string, error) {
	var files []string
	err := appfiles.ApplicationFiles{}.WalkAppFiles(path, func(relpath, _ string) error {
		filename := filepath.Base(relpath)
		switch {
		case
			regexp.MustCompile(`^.+\.droplet$`).MatchString(filename),
			regexp.MustCompile(`^\..+\.cache$`).MatchString(filename):
			return nil
		}
		files = append(files, relpath)
		return nil
	})
	return files, err
}

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

func (f *FS) IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
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
