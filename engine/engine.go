package engine

import (
	"archive/tar"
	"bytes"
	"io"
)

type Stream struct {
	io.ReadCloser
	Size int64
}

func NewStream(data io.ReadCloser, size int64) Stream {
	return Stream{data, size}
}

func (s Stream) Out(dst io.Writer) error {
	defer s.Close()
	if _, err := io.CopyN(dst, s, s.Size); err != nil {
		return err
	}
	return nil
}

func tarFile(name string, contents io.Reader, size, mode int64) (io.Reader, error) {
	tarBuffer := &bytes.Buffer{}
	tarball := tar.NewWriter(tarBuffer)
	defer tarball.Close()
	header := &tar.Header{Name: name, Size: size, Mode: mode}
	if err := tarball.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := io.CopyN(tarball, contents, size); err != nil {
		return nil, err
	}
	return tarBuffer, nil
}
