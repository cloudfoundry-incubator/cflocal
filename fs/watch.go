// +build !darwin

package fs

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TODO: replace done chan with done func
func (f *FS) Watch(dir string, wait time.Duration) (change <-chan time.Time, done chan<- struct{}, err error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Mode().IsDir() {
			watcher.Add(path) // TODO: log error
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}

	out := make(chan time.Time)
	stop := make(chan struct{})
	go func() {
		var (
			t     time.Time
			send  chan time.Time
			after <-chan time.Time
		)
		for {
			select {
			case <-watcher.Errors: // TODO: log error
			case event := <-watcher.Events:
				if !hasOp(event.Op, fsnotify.Chmod) {
					after = time.After(wait)
				}
			case t = <-after:
				send = out
			case send <- t:
				send = nil
			case <-stop:
				watcher.Close()
				return
			}
		}
	}()

	return out, stop, nil
}

func hasOp(op fsnotify.Op, ops ...fsnotify.Op) bool {
	for _, o := range ops {
		if op&o == o {
			return true
		}
	}
	return false
}
