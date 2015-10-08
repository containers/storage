package fsync

import (
	"testing"
)

func TestLock(t *testing.T) {
	m, err := Get("lockfile")
	if err != nil {
		t.Error(err)
		return
	}
	m.RLock()
	m.RUnlock()
	m.Lock()
	if !m.Updated() {
		t.Errorf("lock appears to have been updated by us?")
		return
	}
	m.Touch()
	if m.Updated() {
		t.Errorf("lock appears to have been updated by someone else")
		return
	}
	m.Unlock()
	r := m.RLocker()
	r.Lock()
	r.Unlock()
}
