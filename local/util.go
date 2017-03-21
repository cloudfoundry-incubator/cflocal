package local

import (
	"encoding/json"
	"errors"
	"io"
)

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

func checkBody(body io.Reader) error {
	decoder := json.NewDecoder(body)
	for {
		var stream struct{ Error string }
		if err := decoder.Decode(&stream); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if stream.Error != "" {
			return errors.New(stream.Error)
		}
	}
}
