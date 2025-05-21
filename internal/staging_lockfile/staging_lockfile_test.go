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

// subTouchMain is a child process which opens the lock file, closes stdout to
// indicate that it has acquired the lock, waits for stdin to get closed,
// updates the last-writer for the lockfile, and then unlocks the file.
func subTouchMain() {
	if len(os.Args) != 2 {
		logrus.Fatalf("expected two args, got %d", len(os.Args))
	}
	tf, err := GetLockFile(os.Args[1])
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

// subLockMain is a child process which opens the lock file, closes stdout to
// indicate that it has acquired the lock, waits for stdin to get closed, and
// then unlocks the file.
func subLockMain() {
	if len(os.Args) != 2 {
		logrus.Fatalf("expected two args, got %d", len(os.Args))
	}
	tf, err := GetLockFile(os.Args[1])
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
func subLock(l *namedLockFile) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
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

// subRLockMain is a child process which opens the lock file, closes stdout to
// indicate that it has acquired the read lock, waits for stdin to get closed,
// and then unlocks the file.
func subRLockMain() {
	if len(os.Args) != 2 {
		logrus.Fatalf("expected two args, got %d", len(os.Args))
	}
	tf, err := GetLockFile(os.Args[1])
	if err != nil {
		logrus.Fatalf("error opening lock file %q: %v", os.Args[1], err)
	}
	tf.RLock()
	os.Stdout.Close()
	_, err = io.Copy(io.Discard, os.Stdin)
	if err != nil {
		logrus.Fatalf("error reading stdin: %v", err)
	}
	tf.Unlock()
}

// subRLock starts a child process.  If it doesn't return an error, the caller
// should wait for the first ReadCloser by reading it until it receives an EOF.
// At that point, the child will have acquired a read lock.  It can then signal
// that the child should release the lock by closing the WriteCloser.
// The caller must call Wait() on the returned cmd.
func subRLock(l *namedLockFile) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
	cmd := reexec.Command("subRLock", l.name)
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
	reexec.Register("subTouch", subTouchMain)
	reexec.Register("subRLock", subRLockMain)
	reexec.Register("subLock", subLockMain)
}

type namedLockFile struct {
	*LockFile
	name string
}

func getNamedLockFile(ro bool) (*namedLockFile, error) {
	var l *LockFile
	tf, err := os.CreateTemp("", "lockfile")
	if err != nil {
		return nil, err
	}
	name := tf.Name()
	tf.Close()
	if ro {
		l, err = GetROLockFile(name)
	} else {
		l, err = GetLockFile(name)
	}
	if err != nil {
		return nil, err
	}
	return &namedLockFile{LockFile: l, name: name}, nil
}

func getTempLockfile() (*namedLockFile, error) {
	return getNamedLockFile(false)
}

func getTempROLockfile() (*namedLockFile, error) {
	return getNamedLockFile(true)
}

func TestLockfileName(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")

	// l.Assert* are NOT usable for determining lock state if there are concurrent users of the lock.
	// It’s just about acceptable for these smoke tests.
	assert.Panics(t, l.AssertLocked)

	l.RLock()
	l.AssertLocked()
	assert.Panics(t, l.AssertLockedForWriting)
	l.Unlock()

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")

	l.Lock()
	l.AssertLocked()
	l.AssertLockedForWriting()
	l.Unlock()

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")
}

func TestTryWriteLockFile(t *testing.T) {
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
		errChan <- l.TryRLock()
		errChan <- l.TryLock()
	}()
	assert.NotNil(t, <-errChan)
	assert.NotNil(t, <-errChan)

	l.Unlock()
}

func TestTryReadLockFile(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	err = l.TryRLock()
	assert.Nil(t, err)

	l.AssertLocked()

	errChan := make(chan error)
	go func() {
		errChan <- l.TryRLock()
		l.Unlock()

		errChan <- l.TryLock()
	}()
	assert.Nil(t, <-errChan)
	assert.NotNil(t, <-errChan)

	l.Unlock()

	go func() {
		errChan <- l.TryLock()
		l.Unlock()
	}()
	assert.Nil(t, <-errChan)
}

func TestLockfileRead(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	l.RLock()
	l.AssertLocked()
	assert.Panics(t, l.AssertLockedForWriting)
	l.Unlock()
}

func TestROLockfileRead(t *testing.T) {
	l, err := getTempROLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	l.RLock()
	l.AssertLocked()
	assert.Panics(t, l.AssertLockedForWriting)
	l.Unlock()
}

func TestLockfileWrite(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	l.Lock()
	l.AssertLocked()
	l.AssertLockedForWriting()
	l.Unlock()
}

func TestROLockfileWrite(t *testing.T) {
	l, err := getTempROLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	defer func() {
		assert.NotNil(t, recover(), "Should have panicked trying to take a write lock using a read lock")
	}()
	l.Lock()
	l.AssertLocked()
	assert.Panics(t, l.AssertLockedForWriting)
	l.Unlock()
}

func TestLockfileWriteConcurrent(t *testing.T) {
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

func TestLockfileReadConcurrent(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	// the test below is inspired by the stdlib's rwmutex tests
	numReaders := 1000
	locked := make(chan bool)
	unlocked := make(chan bool)
	done := make(chan bool)

	for range numReaders {
		go func() {
			l.RLock()
			locked <- true
			<-unlocked
			l.Unlock()
			done <- true
		}()
	}

	// Wait for all parallel locks to succeed
	for range numReaders {
		<-locked
	}
	// Instruct all parallel locks to unlock
	for range numReaders {
		unlocked <- true
	}
	// Wait for all parallel locks to be unlocked
	for range numReaders {
		<-done
	}
}

func TestLockfileMixedConcurrent(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	counter := int32(0)
	diff := int32(10000)
	numIterations := 10
	numReaders := 100
	numWriters := 50

	done := make(chan bool)

	// A writer always adds `diff` to the counter. Hence, `diff` is the
	// only valid value in the critical section.
	writer := func(c *int32) {
		for range numIterations {
			l.Lock()
			tmp := atomic.AddInt32(c, diff)
			assert.True(t, tmp == diff, "counter should be %d but instead is %d", diff, tmp)
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(c, diff*(-1))
			l.Unlock()
		}
		done <- true
	}

	// A reader always adds `1` to the counter. Hence,
	// [1,`numReaders*numIterations`] are valid values.
	reader := func(c *int32) {
		for range numIterations {
			l.RLock()
			tmp := atomic.AddInt32(c, 1)
			assert.True(t, tmp >= 1 && tmp < diff)
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(c, -1)
			l.Unlock()
		}
		done <- true
	}

	for i := range numReaders {
		go reader(&counter)
		// schedule a writer every 2nd iteration
		if i%2 == 1 {
			go writer(&counter)
		}
	}

	for range numReaders + numWriters {
		<-done
	}
}

func TestLockfileMultiprocessRead(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var rcounter, rhighest int64
	var highestMutex sync.Mutex
	subs := make([]struct {
		cmd    *exec.Cmd
		stdin  io.WriteCloser
		stdout io.ReadCloser
	}, 100)
	for i := range subs {
		cmd, stdin, stdout, err := subRLock(l)
		require.Nil(t, err, "error starting subprocess %d to take a read lock", i+1)
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
				t.Logf("\tchild %4d acquired the read lock\n", i+1)
			}
			workingRcounter := atomic.AddInt64(&rcounter, 1)
			highestMutex.Lock()
			if workingRcounter > rhighest {
				rhighest = workingRcounter
			}
			highestMutex.Unlock()
			time.Sleep(1 * time.Second)
			atomic.AddInt64(&rcounter, -1)
			if testing.Verbose() {
				t.Logf("\ttelling child %4d to release the read lock\n", i+1)
			}
			subs[i].stdin.Close()
			err = subs[i].cmd.Wait()
			require.NoError(t, err)
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.True(t, rhighest > 1, "expected to have multiple reader locks at least once, only had %d", rhighest)
}

func TestLockfileMultiprocessWrite(t *testing.T) {
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

func TestLockfileMultiprocessMixed(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var rcounter, wcounter, rhighest, whighest int64
	var rhighestMutex, whighestMutex sync.Mutex

	const (
		biasP  = 1
		biasQ  = 10
		groups = 15
	)

	writer := func(i int) bool { return (i % biasQ) < biasP }
	subs := make([]struct {
		cmd    *exec.Cmd
		stdin  io.WriteCloser
		stdout io.ReadCloser
	}, biasQ*groups)
	for i := range subs {
		var cmd *exec.Cmd
		var stdin io.WriteCloser
		var stdout io.ReadCloser
		if writer(i) {
			cmd, stdin, stdout, err = subLock(l)
			require.Nil(t, err, "error starting subprocess %d to take a write lock", i+1)
		} else {
			cmd, stdin, stdout, err = subRLock(l)
			require.Nil(t, err, "error starting subprocess %d to take a read lock", i+1)
		}
		subs[i].cmd = cmd
		subs[i].stdin = stdin
		subs[i].stdout = stdout
	}
	for i := range subs {
		wg.Add(1)
		go func(i int) {
			// wait for the child to acquire whatever lock it wants
			_, err := io.Copy(io.Discard, subs[i].stdout)
			require.NoError(t, err)
			if writer(i) {
				// child acquired a write lock
				if testing.Verbose() {
					t.Logf("\tchild %4d acquired the write lock\n", i+1)
				}
				workingWcounter := atomic.AddInt64(&wcounter, 1)
				whighestMutex.Lock()
				if workingWcounter > whighest {
					whighest = workingWcounter
				}
				workingRcounter := atomic.LoadInt64(&rcounter)
				require.Zero(t, workingRcounter, "acquired a write lock while we appear to have read locks")
				whighestMutex.Unlock()
			} else {
				// child acquired a read lock
				if testing.Verbose() {
					t.Logf("\tchild %4d acquired the read lock\n", i+1)
				}
				workingRcounter := atomic.AddInt64(&rcounter, 1)
				rhighestMutex.Lock()
				if workingRcounter > rhighest {
					rhighest = workingRcounter
				}
				workingWcounter := atomic.LoadInt64(&wcounter)
				require.Zero(t, workingWcounter, "acquired a read lock while we appear to have write locks")
				rhighestMutex.Unlock()
			}
			time.Sleep(1 * time.Second)
			if writer(i) {
				atomic.AddInt64(&wcounter, -1)
				if testing.Verbose() {
					t.Logf("\ttelling child %4d to release the write lock\n", i+1)
				}
			} else {
				atomic.AddInt64(&rcounter, -1)
				if testing.Verbose() {
					t.Logf("\ttelling child %4d to release the read lock\n", i+1)
				}
			}
			subs[i].stdin.Close()
			err = subs[i].cmd.Wait()
			require.NoError(t, err)
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.True(t, rhighest > 1, "expected to have more than one reader lock active at a time at least once, only had %d", rhighest)
	assert.True(t, whighest == 1, "expected to have no more than one writer lock active at a time, had %d", whighest)
}

func TestOpenLock(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		prepare func() (path string, readOnly bool)
	}{
		{
			name: "file exists (read/write)",
			prepare: func() (string, bool) {
				tempFile, err := os.CreateTemp("", "lock-")
				require.NoError(t, err)
				tempFile.Close()
				return tempFile.Name(), false
			},
		},
		{
			name: "file exists readonly (readonly)",
			prepare: func() (string, bool) {
				tempFile, err := os.CreateTemp("", "lock-")
				require.NoError(t, err)
				tempFile.Close()
				return tempFile.Name(), true
			},
		},
		{
			name: "base dir exists (read/write)",
			prepare: func() (string, bool) {
				tempDir := os.TempDir()
				require.DirExists(t, tempDir)
				return filepath.Join(tempDir, "test-1.lock"), false
			},
		},
		{
			name: "base dir not exists (read/write)",
			prepare: func() (string, bool) {
				tempDir, err := os.MkdirTemp("", "lock-")
				require.NoError(t, err)
				return filepath.Join(tempDir, "subdir", "test-1.lock"), false
			},
		},
	} {
		path, readOnly := tc.prepare()

		fd, err := openLock(path, readOnly)
		require.NoError(t, err, tc.name)
		unlockAndCloseHandle(fd)

		fd, err = openLock(path, readOnly)
		require.NoError(t, err)
		unlockAndCloseHandle(fd)

		require.Nil(t, os.RemoveAll(path))
	}
}
