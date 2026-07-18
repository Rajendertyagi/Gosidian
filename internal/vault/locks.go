package vault

import "sync"

// LockPath acquires an exclusive in-process lock for the canonical form of
// the given vault-relative path and returns the unlock func. Callers wrap
// their whole load→validate→save sequence (optimistic-locking checks
// included) so that if_match/If-Match behaves as a true compare-and-swap
// instead of a check-then-write race between concurrent agents.
//
// The lock only serializes writers inside this process (MCP tools and the
// HTTP API share the same Vault); external writers — git sync, direct file
// edits on the host — are out of scope. Entries are never evicted: the key
// set is bounded by the number of distinct note paths and a bare mutex is
// tiny, so eviction bookkeeping would cost more than it saves.
func (v *Vault) LockPath(path string) (unlock func()) {
	key := path
	if r, err := v.Rel(path); err == nil {
		key = r
	}
	mu, _ := v.locks.LoadOrStore(key, &sync.Mutex{})
	m := mu.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}
