package vault

import (
	"sync"
	"testing"
	"time"
)

func TestLockPath_SerializesSamePath(t *testing.T) {
	v := New(t.TempDir())
	const n = 32
	counter := 0
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := v.LockPath("proj/note.md")
			defer unlock()
			counter++ // unsynchronized without the lock — -race would flag it
		}()
	}
	wg.Wait()
	if counter != n {
		t.Fatalf("counter = %d, want %d", counter, n)
	}
}

func TestLockPath_CanonicalizesKey(t *testing.T) {
	v := New(t.TempDir())
	unlock := v.LockPath("a.md")
	acquired := make(chan struct{})
	go func() {
		u := v.LockPath("./a.md") // same canonical path, different spelling
		close(acquired)
		u()
	}()
	select {
	case <-acquired:
		t.Fatal("second LockPath acquired while first was still held")
	case <-time.After(50 * time.Millisecond):
	}
	unlock()
	select {
	case <-acquired:
	case <-time.After(2 * time.Second):
		t.Fatal("second LockPath never acquired after unlock")
	}
}

func TestLockPath_IndependentPaths(t *testing.T) {
	v := New(t.TempDir())
	unlockA := v.LockPath("a.md")
	defer unlockA()
	done := make(chan struct{})
	go func() {
		u := v.LockPath("b.md")
		u()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("locking a different path blocked behind an unrelated lock")
	}
}
