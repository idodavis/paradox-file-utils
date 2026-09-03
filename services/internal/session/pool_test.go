// pool_test.go verifies that concurrent EnsureSession calls for one id build the
// session exactly once (singleflight) and all callers share it.

package session

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestEnsureSession(t *testing.T) {
	var builds int32
	var events []string
	pool := NewPool(func(id string) (*Session, error) {
		atomic.AddInt32(&builds, 1)
		return NewWithLoc(id, "ck3", "english", nil, nil, nil), nil
	}, func(event string, _ any) { events = append(events, event) })

	const n = 20
	var wg sync.WaitGroup
	sessions := make([]*Session, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, err := pool.EnsureSession("ws")
			if err != nil {
				t.Errorf("EnsureSession: %v", err)
				return
			}
			sessions[i] = s
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt32(&builds); got != 1 {
		t.Fatalf("builds = %d, want 1", got)
	}
	for i := 1; i < n; i++ {
		if sessions[i] != sessions[0] {
			t.Fatalf("caller %d got a different session instance", i)
		}
	}
	if len(events) != 1 || events[0] != EventReady {
		t.Errorf("events = %v, want [%s]", events, EventReady)
	}
}

func TestEnsureSessionEmptyIDDoesNotDrop(t *testing.T) {
	pool := NewPool(func(id string) (*Session, error) {
		return NewWithLoc(id, "ck3", "english", nil, nil, nil), nil
	}, nil)
	live, err := pool.EnsureSession("ws")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.EnsureSession(""); err == nil {
		t.Fatal("empty id: want error")
	}
	if pool.Get("ws") != live {
		t.Fatal("empty EnsureSession dropped the live session")
	}
}

func TestEnsureSessionReturnsExistingBeforeDrop(t *testing.T) {
	var builds int32
	pool := NewPool(func(id string) (*Session, error) {
		atomic.AddInt32(&builds, 1)
		return NewWithLoc(id, "ck3", "english", nil, nil, nil), nil
	}, nil)
	first, err := pool.EnsureSession("ws")
	if err != nil {
		t.Fatal(err)
	}
	second, err := pool.EnsureSession("ws")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("second EnsureSession rebuilt the live session")
	}
	if atomic.LoadInt32(&builds) != 1 {
		t.Fatalf("builds = %d, want 1", atomic.LoadInt32(&builds))
	}
}

func TestDropAll(t *testing.T) {
	pool := NewPool(func(id string) (*Session, error) {
		return NewWithLoc(id, "ck3", "english", nil, nil, nil), nil
	}, nil)
	if _, err := pool.EnsureSession("ws"); err != nil {
		t.Fatal(err)
	}
	pool.DropAll()
	if pool.Get("ws") != nil {
		t.Fatal("DropAll left a session")
	}
}
