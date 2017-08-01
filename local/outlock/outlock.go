package outlock

import (
	"io"
	"io/ioutil"
	"sync"
)

type Writer struct {
	w io.Writer
	s sync.Mutex
}

func New(w io.Writer) *Writer {
	return &Writer{w: w, s: sync.Mutex{}}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	w.s.Lock()
	defer w.s.Unlock()
	return w.w.Write(p)
}

func (w *Writer) Disable() {
	w.s.Lock()
	defer w.s.Unlock()
	w.w = ioutil.Discard
}
