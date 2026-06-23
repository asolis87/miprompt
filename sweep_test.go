package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSweepNodeCaches verifies the TTL sweep removes only stale node caches and
// leaves fresh ones and unrelated files untouched.
func TestSweepNodeCaches(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)

	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)

	write := func(name string, age time.Duration) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("20.0.0\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		mod := now.Add(-age)
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatal(err)
		}
		return p
	}

	fresh := write(nodeCachePrefix+"fresh", 1*time.Hour)    // young: keep
	stale := write(nodeCachePrefix+"stale", 48*time.Hour)   // old: remove
	other := write("unrelated-file", 48*time.Hour)          // not ours: keep

	sweepNodeCaches(24*time.Hour, now)

	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh cache was removed, should have been kept: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale cache was NOT removed, should have been deleted")
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("unrelated file was removed, should have been kept: %v", err)
	}
}
