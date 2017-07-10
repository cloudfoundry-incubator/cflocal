// +build !darwin

package fs

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (f *FS) Watch(dir string, wait time.Duration) (change <-chan map[string]string, done chan<- struct{}, err error) {
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

	out := make(chan map[string]string)
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
				if event.Op != fsnotify.Chmod { // TODO: explicitly specify
					after = time.After(wait)
				}
			case t = <-after:
				send = out
			case send <- map[string]string{"time": t.Format(time.RFC3339)}:
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
