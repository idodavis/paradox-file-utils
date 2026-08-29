// pool.go owns the workspaceID -> *Session map and builds each session at most once
// even under concurrent callers (singleflight). It emits a ready event so the
// frontend refreshes without polling; the emitter and builder are injected so
// this package stays free of Wails and the workspace layout.

package session

import (
	"golang.org/x/sync/singleflight"
	"sync"
)

// Builder constructs a Session for a workspace id (reads mods, loads the cache).
type Builder func(workspaceID string) (*Session, error)

// Emitter publishes a lifecycle event to the frontend (e.g. "lang:ready").
type Emitter func(event string, data any)

// EventReady fires once a workspace session is built and live.
const EventReady = "lang:ready"

// Pool caches live sessions keyed by workspace id.
type Pool struct {
	build Builder
	emit  Emitter

	mu       sync.RWMutex
	sessions map[string]*Session
	sf       singleflight.Group
}

// NewPool creates a pool. emit may be nil (no events).
func NewPool(build Builder, emit Emitter) *Pool {
	return &Pool{build: build, emit: emit, sessions: map[string]*Session{}}
}

// EnsureSession returns the session for id, building it once if absent. Concurrent
// callers for the same id share a single build.
func (p *Pool) EnsureSession(id string) (*Session, error) {
	p.mu.RLock()
	if s := p.sessions[id]; s != nil {
		p.mu.RUnlock()
		return s, nil
	}
	p.mu.RUnlock()

	v, err, _ := p.sf.Do(id, func() (any, error) {
		s, err := p.build(id)
		if err != nil {
			return nil, err
		}
		p.mu.Lock()
		p.sessions[id] = s
		p.mu.Unlock()
		p.fire(EventReady, id)
		return s, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*Session), nil
}

// Get returns the live session for id, or nil if none is built.
func (p *Pool) Get(id string) *Session {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sessions[id]
}

// Drop discards the session for id (e.g. workspace closed).
func (p *Pool) Drop(id string) {
	p.mu.Lock()
	s := p.sessions[id]
	delete(p.sessions, id)
	p.mu.Unlock()
	if s != nil {
		s.Close()
	}
}

// fire emits an event if an emitter is configured.
func (p *Pool) fire(event, id string) {
	if p.emit != nil {
		p.emit(event, id)
	}
}
