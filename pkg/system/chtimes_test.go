package system

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// prepareTempFile creates a temporary file in a temporary directory.
func prepareTempFile(t *testing.T) string {
	file := filepath.Join(t.TempDir(), "exist")
	if err := ioutil.WriteFile(file, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	return file
}

// TestChtimes tests Chtimes on a tempfile. Test only mTime, because aTime is OS dependent
func TestChtimes(t *testing.T) {
	file := prepareTempFile(t)

	beforeUnixEpochTime := time.Unix(0, 0).Add(-100 * time.Second)
	unixEpochTime := time.Unix(0, 0)
	afterUnixEpochTime := time.Unix(100, 0)
	unixMaxTime := maxTime

	// Test both aTime and mTime set to Unix Epoch
	if err := Chtimes(file, unixEpochTime, unixEpochTime); err != nil {
		t.Fatal(err)
	}

	f, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}

	if f.ModTime() != unixEpochTime {
		t.Fatalf("Expected: %s, got: %s", unixEpochTime, f.ModTime())
	}

	// Test aTime before Unix Epoch and mTime set to Unix Epoch
	if err := Chtimes(file, beforeUnixEpochTime, unixEpochTime); err != nil {
		t.Fatal(err)
	}

	f, err = os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}

	if f.ModTime() != unixEpochTime {
		t.Fatalf("Expected: %s, got: %s", unixEpochTime, f.ModTime())
	}

	// Test aTime set to Unix Epoch and mTime before Unix Epoch
	if err := Chtimes(file, unixEpochTime, beforeUnixEpochTime); err != nil {
		t.Fatal(err)
	}

	f, err = os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}

	if f.ModTime() != unixEpochTime {
		t.Fatalf("Expected: %s, got: %s", unixEpochTime, f.ModTime())
	}

	// Test both aTime and mTime set to after Unix Epoch (valid time)
	if err := Chtimes(file, afterUnixEpochTime, afterUnixEpochTime); err != nil {
		t.Fatal(err)
	}

	f, err = os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}

	if f.ModTime() != afterUnixEpochTime {
		t.Fatalf("Expected: %s, got: %s", afterUnixEpochTime, f.ModTime())
	}

	// Test both aTime and mTime set to Unix max time
	if err := Chtimes(file, unixMaxTime, unixMaxTime); err != nil {
		t.Fatal(err)
	}

	f, err = os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}

	if f.ModTime().Truncate(time.Second) != unixMaxTime.Truncate(time.Second) {
		t.Fatalf("Expected: %s, got: %s", unixMaxTime.Truncate(time.Second), f.ModTime().Truncate(time.Second))
	}
}
