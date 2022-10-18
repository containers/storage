package lockfile

import (
	"io"
	"os"
	"os/exec"
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
	tf, err := GetLockfile(os.Args[1])
	if err != nil {
		logrus.Fatalf("error opening lock file %q: %v", os.Args[1], err)
	}
	tf.Lock()
	os.Stdout.Close()
	_, err = io.Copy(io.Discard, os.Stdin)
	if err != nil {
		logrus.Fatalf("error reading stdin: %v", err)
	}
	err = tf.Touch()
	if err != nil {
		logrus.Fatalf("error touching lock: %v", err)
	}
	tf.Unlock()
}

// subTouch starts a child process.  If it doesn't return an error, the caller
// should wait for the first ReadCloser by reading it until it receives an EOF.
// At that point, the child will have acquired the lock.  It can then signal
// that the child should Touch() the lock by closing the WriteCloser.  The
// second ReadCloser will be closed when the child has finished.
// The caller must call Wait() on the returned cmd.
func subTouch(l *namedLocker) (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	cmd := reexec.Command("subTouch", l.name)
	wc, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	ec, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, nil, err
	}
	return cmd, wc, rc, ec, nil
}

// subLockMain is a child process which opens the lock file, closes stdout to
// indicate that it has acquired the lock, waits for stdin to get closed, and
// then unlocks the file.
func subLockMain() {
	if len(os.Args) != 2 {
		logrus.Fatalf("expected two args, got %d", len(os.Args))
	}
	tf, err := GetLockfile(os.Args[1])
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
func subLock(l *namedLocker) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
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
	tf, err := GetLockfile(os.Args[1])
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
func subRLock(l *namedLocker) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
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

type namedLocker struct {
	Locker
	name string
}

func getNamedLocker(ro bool) (*namedLocker, error) {
	var l Locker
	tf, err := os.CreateTemp("", "lockfile")
	if err != nil {
		return nil, err
	}
	name := tf.Name()
	tf.Close()
	if ro {
		l, err = GetROLockfile(name)
	} else {
		l, err = GetLockfile(name)
	}
	if err != nil {
		return nil, err
	}
	return &namedLocker{Locker: l, name: name}, nil
}

func getTempLockfile() (*namedLocker, error) {
	return getNamedLocker(false)
}

func getTempROLockfile() (*namedLocker, error) {
	return getNamedLocker(true)
}

func TestLockfileName(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")

	assert.False(t, l.Locked(), "Locked() said we have a write lock")

	l.RLock()
	assert.False(t, l.Locked(), "Locked() said we have a write lock")
	l.Unlock()

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")

	l.Lock()
	assert.True(t, l.Locked(), "Locked() said we didn't have a write lock")
	l.Unlock()

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")
}

func TestLockfileRead(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	l.RLock()
	assert.False(t, l.Locked(), "Locked() said we have a write lock")
	l.Unlock()
}

func TestROLockfileRead(t *testing.T) {
	l, err := getTempROLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	l.RLock()
	assert.False(t, l.Locked(), "Locked() said we have a write lock")
	l.Unlock()
}

func TestLockfileWrite(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	l.Lock()
	assert.True(t, l.Locked(), "Locked() said we didn't have a write lock")
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
	assert.False(t, l.Locked(), "Locked() said we have a write lock")
	l.Unlock()
}

func TestLockfileTouch(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	l.Lock()
	m, err := l.Modified()
	require.Nil(t, err, "got an error from Modified()")
	assert.True(t, m, "new lock file does not appear to have changed")

	now := time.Now()
	assert.False(t, l.TouchedSince(now), "file timestamp was updated for no reason")

	time.Sleep(2 * time.Second)
	err = l.Touch()
	require.Nil(t, err, "got an error from Touch()")
	assert.True(t, l.TouchedSince(now), "file timestamp was not updated by Touch()")

	m, err = l.Modified()
	require.Nil(t, err, "got an error from Modified()")
	assert.False(t, m, "lock file mistakenly indicated that someone else has modified it")

	cmd, stdin, stdout, stderr, err := subTouch(l)
	require.Nil(t, err, "got an error starting a subprocess to touch the lockfile")
	l.Unlock()
	_, err = io.Copy(io.Discard, stdout)
	require.NoError(t, err)
	stdin.Close()
	_, err = io.Copy(io.Discard, stderr)
	require.NoError(t, err)
	err = cmd.Wait()
	require.NoError(t, err)
	l.Lock()
	m, err = l.Modified()
	l.Unlock()
	require.Nil(t, err, "got an error from Modified()")
	assert.True(t, m, "lock file failed to notice that someone else modified it")
}

func TestLockfileWriteConcurrent(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var highestMutex sync.Mutex
	var counter, highest int64
	for i := 0; i < 8000; i++ {
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

	for i := 0; i < numReaders; i++ {
		go func() {
			l.RLock()
			locked <- true
			<-unlocked
			l.Unlock()
			done <- true
		}()
	}

	// Wait for all parallel locks to succeed
	for i := 0; i < numReaders; i++ {
		<-locked
	}
	// Instruct all parallel locks to unlock
	for i := 0; i < numReaders; i++ {
		unlocked <- true
	}
	// Wait for all parallel locks to be unlocked
	for i := 0; i < numReaders; i++ {
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
		for i := 0; i < numIterations; i++ {
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
		for i := 0; i < numIterations; i++ {
			l.RLock()
			tmp := atomic.AddInt32(c, 1)
			assert.True(t, tmp >= 1 && tmp < diff)
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(c, -1)
			l.Unlock()
		}
		done <- true
	}

	for i := 0; i < numReaders; i++ {
		go reader(&counter)
		// schedule a writer every 2nd iteration
		if i%2 == 1 {
			go writer(&counter)
		}
	}

	for i := 0; i < numReaders+numWriters; i++ {
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

func TestLockfileMultiprocessModified(t *testing.T) {
	lock, err := getTempLockfile()
	require.NoError(t, err, "creating lock")

	// Lock hasn't been touched yet - initial state.
	lock.Lock()
	modified, err := lock.Modified()
	lock.Unlock()
	assert.NoError(t, err, "checking if lock was modified")
	assert.True(t, modified, "expected Modified() to be true before anyone Touched it")

	lock.Lock()
	modified, err = lock.Modified()
	lock.Unlock()
	assert.NoError(t, err, "checking if lock was modified")
	assert.False(t, modified, "expected Modified() to be false after we check the first time, before anyone Touched it")

	// Take a read lock somewhere, then see if we incorrectly detect changes.
	cmd, wc, rc1, err := subLock(lock)
	wc.Close()
	cmd.Wait()
	rc1.Close()

	lock.Lock()
	modified, err = lock.Modified()
	lock.Unlock()
	assert.NoError(t, err, "checking if lock was modified")
	assert.False(t, modified, "expected Modified() to be false after someone else locked but did not Touch it")

	// Take a write lock somewhere, then see if we correctly detect changes.
	cmd, wc, rc1, rc2, err := subTouch(lock)
	wc.Close()
	cmd.Wait()
	rc1.Close()
	rc2.Close()

	lock.Lock()
	modified, err = lock.Modified()
	lock.Unlock()
	assert.NoError(t, err, "checking if lock was modified")
	assert.True(t, modified, "expected Modified() to be true after someone else Touched it")

	// Take a read lock somewhere, then see if we incorrectly detect changes.
	cmd, wc, rc1, err = subLock(lock)
	wc.Close()
	cmd.Wait()
	rc1.Close()

	lock.Lock()
	modified, err = lock.Modified()
	lock.Unlock()
	assert.NoError(t, err, "checking if lock was modified")
	assert.False(t, modified, "expected Modified() to be false after someone else locked but did not Touch it")
}
