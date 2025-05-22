package staging_lockfile

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/containers/storage/pkg/reexec"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Warning: this is not an exhaustive set of tests.

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

// subLockMain is a child process which opens the lock file, closes stdout to
// indicate that it has acquired the lock, waits for stdin to get closed, and
// then unlocks the file.
func subLockMain() {
	if len(os.Args) != 2 {
		logrus.Fatalf("expected two args, got %d", len(os.Args))
	}
	tf, err := GetStagingLockFile(os.Args[1])
	if err != nil {
		logrus.Fatalf("error opening lock file %q: %v", os.Args[1], err)
	}
	tf.Lock()
	os.Stdout.Close()
	_, err = io.Copy(io.Discard, os.Stdin)
	if err != nil {
		logrus.Fatalf("error reading stdin: %v", err)
	}
	tf.Unlock()
}

// subLock starts a child process.  If it doesn't return an error, the caller
// should wait for the first ReadCloser by reading it until it receives an EOF.
// At that point, the child will have acquired the lock.  It can then signal
// that the child should release the lock by closing the WriteCloser.
// The caller must call Wait() on the returned cmd.
func subLock(l *namedStagingLockFile) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
	cmd := reexec.Command("subLock", l.name)
	wc, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}
	return cmd, wc, rc, nil
}

func init() {
	reexec.Register("subLock", subLockMain)
}

type namedStagingLockFile struct {
	*StagingLockFile
	name string
}

func getNamedStagingLockfile() (*namedStagingLockFile, error) {
	tf, err := os.CreateTemp("", "lockfile")
	if err != nil {
		return nil, err
	}
	name := tf.Name()
	tf.Close()
	l, err := GetStagingLockFile(name)
	if err != nil {
		return nil, err
	}
	return &namedStagingLockFile{StagingLockFile: l, name: name}, nil
}

func getTempLockfile() (*namedStagingLockFile, error) {
	return getNamedStagingLockfile()
}

func TestLockfileName(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")

	// l.Assert* are NOT usable for determining lock state if there are concurrent users of the lock.
	// It’s just about acceptable for these smoke tests.
	assert.Panics(t, l.AssertLocked)

	l.Lock()
	l.AssertLocked()
	l.Unlock()

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")

	l.Lock()
	l.AssertLocked()
	l.Unlock()

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")
}

func TestTryWriteStagingLockfile(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	err = l.TryLock()
	assert.Nil(t, err)

	l.AssertLocked()

	errChan := make(chan error)
	go func() {
		errChan <- l.TryLock()
	}()
	assert.NotNil(t, <-errChan)

	l.Unlock()
}

func TestStagingLockfile(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	l.Lock()
	l.AssertLocked()
	l.Unlock()
}

func TestLockfileConcurrent(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var highestMutex sync.Mutex
	var counter, highest int64
	for range 8000 {
		wg.Add(1)
		go func() {
			l.Lock()
			workingCounter := atomic.AddInt64(&counter, 1)
			assert.True(t, workingCounter >= 0, "counter should never be less than zero")
			highestMutex.Lock()
			if workingCounter > highest {
				// multiple writers should not be able to hold
				// this lock at the same time, so there should
				// be no point at which two goroutines are
				// between the AddInt64() above and the one
				// below
				highest = workingCounter
			}
			highestMutex.Unlock()
			atomic.AddInt64(&counter, -1)
			l.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()
	assert.True(t, highest == 1, "counter should never have gone above 1, got to %d", highest)
}

func TestLockfileMultiProcess(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var wcounter, whighest int64
	var highestMutex sync.Mutex
	subs := make([]struct {
		cmd    *exec.Cmd
		stdin  io.WriteCloser
		stdout io.ReadCloser
	}, 10)
	for i := range subs {
		cmd, stdin, stdout, err := subLock(l)
		require.Nil(t, err, "error starting subprocess %d to take a write lock", i+1)
		subs[i].cmd = cmd
		subs[i].stdin = stdin
		subs[i].stdout = stdout
	}
	for i := range subs {
		wg.Add(1)
		go func(i int) {
			_, err := io.Copy(io.Discard, subs[i].stdout)
			require.NoError(t, err)
			if testing.Verbose() {
				t.Logf("\tchild %4d acquired the write lock\n", i+1)
			}
			workingWcounter := atomic.AddInt64(&wcounter, 1)
			highestMutex.Lock()
			if workingWcounter > whighest {
				whighest = workingWcounter
			}
			highestMutex.Unlock()
			time.Sleep(1 * time.Second)
			atomic.AddInt64(&wcounter, -1)
			if testing.Verbose() {
				t.Logf("\ttelling child %4d to release the write lock\n", i+1)
			}
			subs[i].stdin.Close()
			err = subs[i].cmd.Wait()
			require.NoError(t, err)
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.True(t, whighest == 1, "expected to have no more than one writer lock active at a time, had %d", whighest)
}

func TestOpenLock(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		prepare func() (path string)
	}{
		{
			name: "file exists (read/write)",
			prepare: func() string {
				tempFile, err := os.CreateTemp("", "lock-")
				require.NoError(t, err)
				tempFile.Close()
				return tempFile.Name()
			},
		},
		{
			name: "base dir exists (read/write)",
			prepare: func() string {
				tempDir := os.TempDir()
				require.DirExists(t, tempDir)
				return filepath.Join(tempDir, "test-1.lock")
			},
		},
		{
			name: "base dir not exists (read/write)",
			prepare: func() string {
				tempDir, err := os.MkdirTemp("", "lock-")
				require.NoError(t, err)
				return filepath.Join(tempDir, "subdir", "test-1.lock")
			},
		},
	} {
		path := tc.prepare()

		fd, err := openLock(path)
		require.NoError(t, err, tc.name)
		unlockAndCloseHandle(fd)

		fd, err = openLock(path)
		require.NoError(t, err)
		unlockAndCloseHandle(fd)

		require.Nil(t, os.RemoveAll(path))
	}
}
