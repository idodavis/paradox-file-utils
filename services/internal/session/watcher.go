// watcher.go watches mod roots for EXTERNAL changes (git checkout/pull, patcher or
// merge writes, another editor) and funnels them into the same single reindex path
// as an editor save. Editor saves are already covered by DidSave; the identical-
// bytes check in reindexFileLocked coalesces the two. Events are debounced per path.

package session

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watchDebounce is how long a path must be quiet before its change is reported;
// editors and git often emit several events per logical write.
const watchDebounce = 150 * time.Millisecond

// Watcher reports debounced changes to indexable files under a set of roots.
type Watcher struct {
	fsw      *fsnotify.Watcher
	onChange func(path string)

	mu     sync.Mutex
	timers map[string]*time.Timer
	closed bool
	done   chan struct{}
}

// NewWatcher starts watching roots recursively, calling onChange(absPath) once a
// changed .txt/.gui/.yml file settles. Close stops it.
func NewWatcher(roots []string, onChange func(path string)) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{fsw: fsw, onChange: onChange, timers: map[string]*time.Timer{}, done: make(chan struct{})}
	for _, root := range roots {
		w.addTree(root)
	}
	go w.loop()
	return w, nil
}

// Close stops watching and releases resources. Pending debounce timers are
// stopped so onChange cannot fire after teardown.
func (w *Watcher) Close() error {
	close(w.done)
	w.mu.Lock()
	w.closed = true
	for name, t := range w.timers {
		t.Stop()
		delete(w.timers, name)
	}
	w.mu.Unlock()
	return w.fsw.Close()
}

// addTree adds root and all its subdirectories to the watch set (fsnotify is not
// recursive on its own).
func (w *Watcher) addTree(root string) {
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = w.fsw.Add(p)
		}
		return nil
	})
}

// loop consumes fsnotify events until Close.
func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handle(ev)
		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
		}
	}
}

// handle debounces a relevant event and, for new directories, extends the watch.
func (w *Watcher) handle(ev fsnotify.Event) {
	if ev.Op&fsnotify.Create != 0 {
		if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
			w.addTree(ev.Name)
			return
		}
	}
	if !indexable(ev.Name) {
		return
	}
	w.mu.Lock()
	if t := w.timers[ev.Name]; t != nil {
		t.Stop()
	}
	name := ev.Name
	w.timers[name] = time.AfterFunc(watchDebounce, func() {
		w.mu.Lock()
		delete(w.timers, name)
		closed := w.closed
		w.mu.Unlock()
		if closed {
			return
		}
		w.onChange(name)
	})
	w.mu.Unlock()
}

// indexable reports whether a path is a file kind the index tracks.
func indexable(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".txt") || strings.HasSuffix(lower, ".gui") ||
		strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml")
}
