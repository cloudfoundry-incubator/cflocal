package engine

import "io"

type Stream struct {
	io.ReadCloser
	Size int64
}

func NewStream(data io.ReadCloser, size int64) Stream {
	return Stream{data, size}
}

func (s Stream) Write(dst io.Writer) error {
	if _, err := io.CopyN(dst, s, s.Size); err != nil && err != io.EOF {
		return err
	}
	return nil
}
