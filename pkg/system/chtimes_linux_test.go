//go:build linux

package system

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func atime(t *testing.T, file string) time.Time {
	t.Helper()

	fi, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}

	stat := fi.Sys().(*syscall.Stat_t)
	return time.Unix(stat.Atim.Unix())
}

// TestChtimesLinux tests Chtimes access time on a tempfile on Linux
func TestChtimesLinux(t *testing.T) {
	file := prepareTempFile(t)

	beforeUnixEpochTime := time.Unix(0, 0).Add(-100 * time.Second)
	unixEpochTime := time.Unix(0, 0)
	afterUnixEpochTime := time.Unix(100, 0)
	unixMaxTime := maxTime

	// Test both aTime and mTime set to Unix Epoch
	if err := Chtimes(file, unixEpochTime, unixEpochTime); err != nil {
		t.Fatal(err)
	}

	aTime := atime(t, file)
	if aTime != unixEpochTime {
		t.Fatalf("Expected: %s, got: %s", unixEpochTime, aTime)
	}

	// Test aTime before Unix Epoch and mTime set to Unix Epoch
	if err := Chtimes(file, beforeUnixEpochTime, unixEpochTime); err != nil {
		t.Fatal(err)
	}

	aTime = atime(t, file)
	if aTime != unixEpochTime {
		t.Fatalf("Expected: %s, got: %s", unixEpochTime, aTime)
	}

	// Test aTime set to Unix Epoch and mTime before Unix Epoch
	if err := Chtimes(file, unixEpochTime, beforeUnixEpochTime); err != nil {
		t.Fatal(err)
	}

	aTime = atime(t, file)
	if aTime != unixEpochTime {
		t.Fatalf("Expected: %s, got: %s", unixEpochTime, aTime)
	}

	// Test both aTime and mTime set to after Unix Epoch (valid time)
	if err := Chtimes(file, afterUnixEpochTime, afterUnixEpochTime); err != nil {
		t.Fatal(err)
	}

	aTime = atime(t, file)
	if aTime != afterUnixEpochTime {
		t.Fatalf("Expected: %s, got: %s", afterUnixEpochTime, aTime)
	}

	// Test both aTime and mTime set to Unix max time
	if err := Chtimes(file, unixMaxTime, unixMaxTime); err != nil {
		t.Fatal(err)
	}

	aTime = atime(t, file)
	if aTime.Truncate(time.Second) != unixMaxTime.Truncate(time.Second) {
		t.Fatalf("Expected: %s, got: %s", unixMaxTime.Truncate(time.Second), aTime.Truncate(time.Second))
	}
}
