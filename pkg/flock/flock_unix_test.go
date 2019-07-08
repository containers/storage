// build unix

package flock

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/containers/storage/pkg/reexec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		fmt.Fprintf(os.Stderr, "expected two args, got %d", len(os.Args))
		os.Exit(1)
	}
	tf, err := New(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening lock file %q: %v", os.Args[1], err)
		os.Exit(2)
	}
	if err := tf.Lock(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(3)
	}
	os.Stdout.Close()
	if _, err := io.Copy(ioutil.Discard, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(4)
	}
	if err := tf.Unlock(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(5)
	}
}

// subLockMain is a child process which opens the lock file, closes stdout to
// indicate that it has acquired the lock, waits for stdin to get closed, and
// then unlocks the file.
func subLockMain() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "expected two args, got %d", len(os.Args))
		os.Exit(1)
	}
	tf, err := New(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening lock file %q: %v", os.Args[1], err)
		os.Exit(2)
	}
	if err := tf.Lock(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(3)
	}
	os.Stdout.Close()
	if _, err := io.Copy(ioutil.Discard, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(4)
	}
	if err := tf.Unlock(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(5)
	}
}

// subLock starts a child process.  If it doesn't return an error, the caller
// should wait for the first ReadCloser by reading it until it receives an EOF.
// At that point, the child will have acquired the lock.  It can then signal
// that the child should release the lock by closing the WriteCloser.
func subLock(l *namedFlock) (io.WriteCloser, io.ReadCloser, error) {
	cmd := reexec.Command("subLock", l.name)
	wc, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	go func() {
		if err = cmd.Run(); err != nil {
			slurp, _ := ioutil.ReadAll(rc)
			panic(fmt.Sprintf("error running subLock: %s: %s", err.Error(), string(slurp)))
		}
	}()
	return wc, rc, nil
}

// subRLockMain is a child process which opens the lock file, closes stdout to
// indicate that it has acquired the read lock, waits for stdin to get closed,
// and then unlocks the file.
func subRLockMain() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "expected two args, got %d", len(os.Args))
		os.Exit(1)
	}
	tf, err := New(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening lock file %q: %v", os.Args[1], err)
		os.Exit(2)
	}
	if err := tf.RLock(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(3)
	}
	os.Stdout.Close()
	if _, err := io.Copy(ioutil.Discard, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(4)
	}
	if err := tf.Unlock(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(5)
	}
}

// subRLock starts a child process.  If it doesn't return an error, the caller
// should wait for the first ReadCloser by reading it until it receives an EOF.
// At that point, the child will have acquired a read lock.  It can then signal
// that the child should release the lock by closing the WriteCloser.
func subRLock(l *namedFlock) (io.WriteCloser, io.ReadCloser, error) {
	cmd := reexec.Command("subRLock", l.name)
	wc, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	go func() {
		if err = cmd.Run(); err != nil {
			panic(fmt.Sprintf("error running subRLock: %v", err))
		}
	}()
	return wc, rc, nil
}

// subTryLockMain is a child process which opens the lock file, closes stdout to
// indicate that it has acquired the lock, waits for stdin to get closed, and
// then unlocks the file.
func subTryLockMain() {
	fmt.Fprintf(os.Stdout, "expected two args, got %d", len(os.Args))

	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "expected two args, got %d", len(os.Args))
		os.Exit(1)
	}
	tf, err := New(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening lock file %q: %v", os.Args[1], err)
		os.Exit(2)
	}
	for {
		if acquired, err := tf.TryLock(); err != nil {
			fmt.Fprintf(os.Stderr, "1 %v", err)
			os.Exit(3)
		} else if acquired {
			break
		}
	}
	os.Stdout.Close()
	if _, err := io.Copy(ioutil.Discard, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "2 %v", err)
		os.Exit(4)
	}
	if err := tf.Unlock(); err != nil {
		fmt.Fprintf(os.Stderr, "3 %v", err)
		os.Exit(5)
	}
}

// subTryLock starts a child process.  If it doesn't return an error, the caller
// should wait for the first ReadCloser by reading it until it receives an EOF.
// At that point, the child will have acquired the lock.  It can then signal
// that the child should release the lock by closing the WriteCloser.
func subTryLock(l *namedFlock) (io.WriteCloser, io.ReadCloser, error) {
	cmd := reexec.Command("subTryLock", l.name)
	wc, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	go func() {
		if err = cmd.Run(); err != nil {
			slurp, _ := ioutil.ReadAll(rc)
			panic(fmt.Sprintf("error running subTryLock: %v: %v", err, string(slurp)))
		}
	}()
	return wc, rc, nil
}

func init() {
	reexec.Register("subRLock", subRLockMain)
	reexec.Register("subLock", subLockMain)
	reexec.Register("subTryLock", subTryLockMain)
}

type namedFlock struct {
	Flock
	name string
}

func getTempLockfile() (*namedFlock, error) {
	tf, err := ioutil.TempFile("", "lockfile")
	if err != nil {
		return nil, err
	}
	name := tf.Name()
	tf.Close()

	l, err := New(name)
	if err != nil {
		return nil, err
	}
	return &namedFlock{Flock: &l, name: name}, nil
}

func TestLockfileName(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")

	assert.False(t, l.Locked())

	assert.Nil(t, l.RLock())
	assert.True(t, l.Locked())
	assert.Nil(t, l.Unlock())

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")

	assert.Nil(t, l.Lock())
	assert.True(t, l.Locked())
	assert.Nil(t, l.Unlock())

	assert.NotEmpty(t, l.name, "lockfile name should be recorded correctly")
}

func TestRLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	assert.Nil(t, l.RLock())
	assert.True(t, l.Locked())

	acquired, err := l.TryLock()
	assert.Nil(t, err)
	assert.False(t, acquired)

	assert.Nil(t, l.Unlock())
	assert.False(t, l.Locked())
}

func TestLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	assert.Nil(t, l.Lock())
	assert.True(t, l.Locked())

	acquired, err := l.TryLock()
	assert.Nil(t, err)
	assert.False(t, acquired)

	assert.Nil(t, l.Unlock())
	assert.False(t, l.Locked())
}

func TestTryLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)

	acquired, err := l.TryLock()
	assert.Nil(t, err)
	assert.True(t, acquired)
	assert.True(t, l.Locked())

	acquired, err = l.TryLock()
	assert.Nil(t, err)
	assert.False(t, acquired)

	assert.Nil(t, l.Unlock())
	assert.False(t, l.Locked())
}

func TestConcurrentLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var highestMutex sync.Mutex
	var counter, highest int64
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		go func() {
			assert.Nil(t, l.Lock())
			tmp := atomic.AddInt64(&counter, 1)
			assert.True(t, tmp >= 0, "counter should never be less than zero")
			highestMutex.Lock()
			if tmp > highest {
				// multiple writers should not be able to hold
				// this lock at the same time, so there should
				// be no point at which two goroutines are
				// between the AddInt64() above and the one
				// below
				highest = tmp
			}
			highestMutex.Unlock()
			atomic.AddInt64(&counter, -1)
			assert.Nil(t, l.Unlock())
			wg.Done()
		}()
	}
	wg.Wait()
	assert.True(t, highest == 1, "counter should never have gone above 1, got to %d", highest)
}

func TestConcurrentTryLock(t *testing.T) {
	// It's basically the same test as TestConcurrentLock but only executes the
	// critical section, if we acquired the lock.  It also spawns more
	// goroutines to stress the locks a bit more.
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var highestMutex sync.Mutex
	var counter, highest int64
	for i := 0; i < 10000000; i++ {
		wg.Add(1)
		go func() {
			acquired, err := l.TryLock()
			assert.Nil(t, err)
			if acquired {
				tmp := atomic.AddInt64(&counter, 1)
				assert.True(t, tmp >= 0, "counter should never be less than zero")
				highestMutex.Lock()
				if tmp > highest {
					// multiple writers should not be able to hold
					// this lock at the same time, so there should
					// be no point at which two goroutines are
					// between the AddInt64() above and the one
					// below
					highest = tmp
				}
				highestMutex.Unlock()
				atomic.AddInt64(&counter, -1)
				assert.Nil(t, l.Unlock())
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.True(t, highest == 1, "counter should never have gone above 1, got to %d", highest)
}

func TestConcurrentRLock(t *testing.T) {
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
			assert.Nil(t, l.RLock())
			locked <- true
			<-unlocked
			assert.Nil(t, l.Unlock())
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

func TestConcurrentMixedLockRLock(t *testing.T) {
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
			assert.Nil(t, l.Lock())
			tmp := atomic.AddInt32(c, diff)
			assert.True(t, tmp == diff, "counter should be %d but instead is %d", diff, tmp)
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(c, diff*(-1))
			assert.Nil(t, l.Unlock())
		}
		done <- true
	}

	// A reader always adds `1` to the counter. Hence,
	// [1,`numReaders*numIterations`] are valid values.
	reader := func(c *int32) {
		for i := 0; i < numIterations; i++ {
			assert.Nil(t, l.RLock())
			tmp := atomic.AddInt32(c, 1)
			assert.True(t, tmp >= 1 && tmp < diff)
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(c, -1)
			assert.Nil(t, l.Unlock())
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

func TestMultiprocessRLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var rcounter, rhighest int64
	var highestMutex sync.Mutex
	subs := make([]struct {
		stdin  io.WriteCloser
		stdout io.ReadCloser
	}, 100)
	for i := range subs {
		stdin, stdout, err := subRLock(l)
		require.Nil(t, err, "error starting subprocess %d to take a read lock", i+1)
		subs[i].stdin = stdin
		subs[i].stdout = stdout
	}
	for i := range subs {
		wg.Add(1)
		go func(i int) {
			_, err := io.Copy(ioutil.Discard, subs[i].stdout)
			assert.Nil(t, err)
			if testing.Verbose() {
				fmt.Printf("\tchild %4d acquired the read lock\n", i+1)
			}
			atomic.AddInt64(&rcounter, 1)
			highestMutex.Lock()
			if rcounter > rhighest {
				rhighest = rcounter
			}
			highestMutex.Unlock()
			time.Sleep(1 * time.Second)
			atomic.AddInt64(&rcounter, -1)
			if testing.Verbose() {
				fmt.Printf("\ttelling child %4d to release the read lock\n", i+1)
			}
			subs[i].stdin.Close()
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.True(t, rhighest > 1, "expected to have multiple reader locks at least once, only had %d", rhighest)
}

func TestMultiprocessLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var wcounter, whighest int64
	var highestMutex sync.Mutex
	subs := make([]struct {
		stdin  io.WriteCloser
		stdout io.ReadCloser
	}, 10)
	for i := range subs {
		stdin, stdout, err := subLock(l)
		require.Nil(t, err, "error starting subprocess %d to take a write lock", i+1)
		subs[i].stdin = stdin
		subs[i].stdout = stdout
	}
	for i := range subs {
		wg.Add(1)
		go func(i int) {
			_, err := io.Copy(ioutil.Discard, subs[i].stdout)
			assert.Nil(t, err)
			if testing.Verbose() {
				fmt.Printf("\tchild %4d acquired the write lock\n", i+1)
			}
			atomic.AddInt64(&wcounter, 1)
			highestMutex.Lock()
			if wcounter > whighest {
				whighest = wcounter
			}
			highestMutex.Unlock()
			time.Sleep(1 * time.Second)
			atomic.AddInt64(&wcounter, -1)
			if testing.Verbose() {
				fmt.Printf("\ttelling child %4d to release the write lock\n", i+1)
			}
			subs[i].stdin.Close()
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.True(t, whighest == 1, "expected to have no more than one writer lock active at a time, had %d", whighest)
}

func TestMultiprocessTryLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var wcounter, whighest int64
	var highestMutex sync.Mutex
	subs := make([]struct {
		stdin  io.WriteCloser
		stdout io.ReadCloser
	}, 10)
	for i := range subs {
		stdin, stdout, err := subTryLock(l)
		require.Nil(t, err, "error starting subprocess %d to take a write lock", i+1)
		subs[i].stdin = stdin
		subs[i].stdout = stdout
	}
	for i := range subs {
		wg.Add(1)
		go func(i int) {
			_, err := io.Copy(ioutil.Discard, subs[i].stdout)
			assert.Nil(t, err)
			if testing.Verbose() {
				fmt.Printf("\tchild %4d acquired the write lock\n", i+1)
			}
			atomic.AddInt64(&wcounter, 1)
			highestMutex.Lock()
			if wcounter > whighest {
				whighest = wcounter
			}
			highestMutex.Unlock()
			time.Sleep(1 * time.Second)
			atomic.AddInt64(&wcounter, -1)
			if testing.Verbose() {
				fmt.Printf("\ttelling child %4d to release the write lock\n", i+1)
			}
			subs[i].stdin.Close()
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.True(t, whighest == 1, "expected to have no more than one writer lock active at a time, had %d", whighest)
}

func TestMultiprocessLockTryLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var wcounter, whighest int64
	var highestMutex sync.Mutex
	subs := make([]struct {
		stdin  io.WriteCloser
		stdout io.ReadCloser
	}, 10)
	for i := range subs {
		if i%2 == 0 {
			stdin, stdout, err := subLock(l)
			require.Nil(t, err, "error starting subprocess %d to take a write via Lock", i+1)
			subs[i].stdin = stdin
			subs[i].stdout = stdout
		} else {
			stdin, stdout, err := subTryLock(l)
			require.Nil(t, err, "error starting subprocess %d to take a write via TryLock", i+1)
			subs[i].stdin = stdin
			subs[i].stdout = stdout
		}
	}
	for i := range subs {
		wg.Add(1)
		go func(i int) {
			_, err := io.Copy(ioutil.Discard, subs[i].stdout)
			assert.Nil(t, err)
			if testing.Verbose() {
				fmt.Printf("\tchild %4d acquired the write lock\n", i+1)
			}
			atomic.AddInt64(&wcounter, 1)
			highestMutex.Lock()
			if wcounter > whighest {
				whighest = wcounter
			}
			highestMutex.Unlock()
			time.Sleep(1 * time.Second)
			atomic.AddInt64(&wcounter, -1)
			if testing.Verbose() {
				fmt.Printf("\ttelling child %4d to release the write lock\n", i+1)
			}
			subs[i].stdin.Close()
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.True(t, whighest == 1, "expected to have no more than one writer lock active at a time, had %d", whighest)
}

func TestMultiprocessLockRLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var rcounter, wcounter, rhighest, whighest int64
	var rhighestMutex, whighestMutex sync.Mutex
	biasP := 1
	biasQ := 10
	groups := 15
	writer := func(i int) bool { return (i % biasQ) < biasP }
	subs := make([]struct {
		stdin  io.WriteCloser
		stdout io.ReadCloser
	}, biasQ*groups)
	for i := range subs {
		var stdin io.WriteCloser
		var stdout io.ReadCloser
		if writer(i) {
			stdin, stdout, err = subLock(l)
			require.Nil(t, err, "error starting subprocess %d to take a write lock", i+1)
		} else {
			stdin, stdout, err = subRLock(l)
			require.Nil(t, err, "error starting subprocess %d to take a read lock", i+1)
		}
		subs[i].stdin = stdin
		subs[i].stdout = stdout
	}
	for i := range subs {
		wg.Add(1)
		go func(i int) {
			// wait for the child to acquire whatever lock it wants
			_, err := io.Copy(ioutil.Discard, subs[i].stdout)
			assert.Nil(t, err)
			if writer(i) {
				// child acquired a write lock
				if testing.Verbose() {
					fmt.Printf("\tchild %4d acquired the write lock\n", i+1)
				}
				atomic.AddInt64(&wcounter, 1)
				whighestMutex.Lock()
				if wcounter > whighest {
					whighest = wcounter
				}
				require.Zero(t, rcounter, "acquired a write lock while we appear to have read locks")
				whighestMutex.Unlock()
			} else {
				// child acquired a read lock
				if testing.Verbose() {
					fmt.Printf("\tchild %4d acquired the read lock\n", i+1)
				}
				atomic.AddInt64(&rcounter, 1)
				rhighestMutex.Lock()
				if rcounter > rhighest {
					rhighest = rcounter
				}
				require.Zero(t, wcounter, "acquired a read lock while we appear to have write locks")
				rhighestMutex.Unlock()
			}
			time.Sleep(1 * time.Second)
			if writer(i) {
				atomic.AddInt64(&wcounter, -1)
				if testing.Verbose() {
					fmt.Printf("\ttelling child %4d to release the write lock\n", i+1)
				}
			} else {
				atomic.AddInt64(&rcounter, -1)
				if testing.Verbose() {
					fmt.Printf("\ttelling child %4d to release the read lock\n", i+1)
				}
			}
			subs[i].stdin.Close()
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.True(t, rhighest > 1, "expected to have more than one reader lock active at a time at least once, only had %d", rhighest)
	assert.True(t, whighest == 1, "expected to have no more than one writer lock active at a time, had %d", whighest)
}

func TestMultiprocessTryLockRLock(t *testing.T) {
	l, err := getTempLockfile()
	require.Nil(t, err, "error getting temporary lock file")
	defer os.Remove(l.name)
	var wg sync.WaitGroup
	var rcounter, wcounter, rhighest, whighest int64
	var rhighestMutex, whighestMutex sync.Mutex
	biasP := 1
	biasQ := 10
	groups := 15
	writer := func(i int) bool { return (i % biasQ) < biasP }
	subs := make([]struct {
		stdin  io.WriteCloser
		stdout io.ReadCloser
	}, biasQ*groups)
	for i := range subs {
		var stdin io.WriteCloser
		var stdout io.ReadCloser
		if writer(i) {
			stdin, stdout, err = subTryLock(l)
			require.Nil(t, err, "error starting subprocess %d to take a write lock", i+1)
		} else {
			stdin, stdout, err = subRLock(l)
			require.Nil(t, err, "error starting subprocess %d to take a read lock", i+1)
		}
		subs[i].stdin = stdin
		subs[i].stdout = stdout
	}
	for i := range subs {
		wg.Add(1)
		go func(i int) {
			// wait for the child to acquire whatever lock it wants
			_, err := io.Copy(ioutil.Discard, subs[i].stdout)
			assert.Nil(t, err)
			if writer(i) {
				// child acquired a write lock
				if testing.Verbose() {
					fmt.Printf("\tchild %4d acquired the write lock\n", i+1)
				}
				atomic.AddInt64(&wcounter, 1)
				whighestMutex.Lock()
				if wcounter > whighest {
					whighest = wcounter
				}
				require.Zero(t, rcounter, "acquired a write lock while we appear to have read locks")
				whighestMutex.Unlock()
			} else {
				// child acquired a read lock
				if testing.Verbose() {
					fmt.Printf("\tchild %4d acquired the read lock\n", i+1)
				}
				atomic.AddInt64(&rcounter, 1)
				rhighestMutex.Lock()
				if rcounter > rhighest {
					rhighest = rcounter
				}
				require.Zero(t, wcounter, "acquired a read lock while we appear to have write locks")
				rhighestMutex.Unlock()
			}
			time.Sleep(1 * time.Second)
			if writer(i) {
				atomic.AddInt64(&wcounter, -1)
				if testing.Verbose() {
					fmt.Printf("\ttelling child %4d to release the write lock\n", i+1)
				}
			} else {
				atomic.AddInt64(&rcounter, -1)
				if testing.Verbose() {
					fmt.Printf("\ttelling child %4d to release the read lock\n", i+1)
				}
			}
			subs[i].stdin.Close()
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.True(t, rhighest > 1, "expected to have more than one reader lock active at a time at least once, only had %d", rhighest)
	assert.True(t, whighest == 1, "expected to have no more than one writer lock active at a time, had %d", whighest)
}
