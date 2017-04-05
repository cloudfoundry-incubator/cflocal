package engine

import "errors"

type progressMsg string

func (p progressMsg) Err() error {
	return nil
}

func (p progressMsg) Status() string {
	return string(p)
}

type progressNA struct{}

func (p progressNA) Err() error {
	return nil
}

func (p progressNA) Status() string {
	return "N/A"
}

type progressError struct{ error }

func (p progressError) Err() error {
	return p.error
}

func (p progressError) Status() string {
	return p.Error()
}

type progressErrorString string

func (p progressErrorString) Err() error {
	return errors.New(string(p))
}

func (p progressErrorString) Status() string {
	return string(p)
}
