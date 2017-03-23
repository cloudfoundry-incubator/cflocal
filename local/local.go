package local

import (
	"encoding/json"
	"errors"
	"io"
)

type UI interface {
	Loading(message string, f func() error) error
}

type Colorizer func(string, ...interface{}) string

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

type vcapApplication struct {
	ApplicationID      string          `json:"application_id"`
	ApplicationName    string          `json:"application_name"`
	ApplicationURIs    []string        `json:"application_uris"`
	ApplicationVersion string          `json:"application_version"`
	Host               string          `json:"host,omitempty"`
	InstanceID         string          `json:"instance_id,omitempty"`
	InstanceIndex      *uint           `json:"instance_index,omitempty"`
	Limits             map[string]uint `json:"limits"`
	Name               string          `json:"name"`
	Port               *uint           `json:"port,omitempty"`
	SpaceID            string          `json:"space_id"`
	SpaceName          string          `json:"space_name"`
	URIs               []string        `json:"uris"`
	Version            string          `json:"version"`
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
