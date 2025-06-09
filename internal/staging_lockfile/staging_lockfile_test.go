package staging_lockfile

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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

// subTryLockPath starts a child process.
// The caller must call Wait() on the returned cmd.
func subTryLockPath(path string) (*exec.Cmd, io.ReadCloser, error) {
	cmd := reexec.Command("subTryLockPath", path)
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, rc, nil
}

// subTryLockPathMain is a child process which tries to opens the StagingLockfile,
// If it has acquired the lock, it will unlock and delete the file.
// Otherwise, it will print an error message to stdout.
func subTryLockPathMain() {
	if len(os.Args) != 2 {
		fmt.Printf("expected two args, got %d", len(os.Args))
		os.Exit(1)
	}
	tf, err := TryLockPath(os.Args[1])
	if err != nil {
		fmt.Printf("error opening lock file %q: %v", os.Args[1], err)
		os.Exit(1)
	}
	if err := tf.UnlockAndDelete(); err != nil {
		fmt.Printf("error unlocking and deleting lock file %q: %v", os.Args[1], err)
		os.Exit(1)
	}
}

func init() {
	reexec.Register("subTryLockPath", subTryLockPathMain)
}

func TestCreateAndLock(t *testing.T) {
	l, _, err := CreateAndLock(t.TempDir(), "staging-lockfile")
	require.NoError(t, err)

	require.NoError(t, l.UnlockAndDelete())

	require.Empty(t, l.file)
	require.Len(t, stagingLockFiles, 0)
}

func TestTryLockPath(t *testing.T) {
	lockFilePath := filepath.Join(t.TempDir(), "test-staging-lockfile")
	l, err := TryLockPath(lockFilePath)
	require.NoError(t, err)

	require.NoError(t, l.UnlockAndDelete())

	require.Len(t, stagingLockFiles, 0)
	_, err = os.Stat(lockFilePath)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCreateAndLockAndTryLock(t *testing.T) {
	tmpDirPath := t.TempDir()
	l, path, err := CreateAndLock(tmpDirPath, "locktest")
	require.NoError(t, err)
	fullPath := filepath.Join(tmpDirPath, path)
	defer os.Remove(fullPath)

	_, err = TryLockPath(fullPath)
	require.Error(t, err)

	require.NoError(t, l.UnlockAndDelete())
	require.Len(t, stagingLockFiles, 0)

	l2, err := TryLockPath(fullPath)
	require.NoError(t, err)
	require.NoError(t, l2.UnlockAndDelete())

	require.Len(t, stagingLockFiles, 0)
}

func TestUnlockAndDeleteTwice(t *testing.T) {
	tmpDirPath := t.TempDir()
	l, path, err := CreateAndLock(tmpDirPath, "panic-unlockdelete")
	require.NoError(t, err)
	fullPath := filepath.Join(tmpDirPath, path)
	defer os.Remove(fullPath)
	require.NoError(t, l.UnlockAndDelete())
	assert.Panics(t, func() { _ = l.UnlockAndDelete() }, "UnlockAndDelete should panic if not locked")
}

func TestLockFileRecreation(t *testing.T) {
	tmpDirPath := t.TempDir()
	l, path, err := CreateAndLock(tmpDirPath, "recreate-lock")
	require.NoError(t, err)
	require.NoError(t, l.UnlockAndDelete())
	fullPath := filepath.Join(tmpDirPath, path)

	l2, err := TryLockPath(fullPath)
	require.NoError(t, err)
	require.NoError(t, l2.UnlockAndDelete())

	require.Len(t, stagingLockFiles, 0)
}

func TestConcurrentLocking(t *testing.T) {
	const n = 10
	ch := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		go func() {
			l, _, err := CreateAndLock(t.TempDir(), "concurrent-lock")
			require.NoError(t, err)
			require.NoError(t, l.UnlockAndDelete())
			ch <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-ch
	}
	require.Len(t, stagingLockFiles, 0)
}

func TestTryLockPathMultiProcess(t *testing.T) {
	tmpDirPath := t.TempDir()
	lockfile, path, err := CreateAndLock(tmpDirPath, "test-staging-lockfile")
	require.NoError(t, err)
	fullPath := filepath.Join(tmpDirPath, path)

	expectedErrMsg := fmt.Sprintf("error opening lock file %q: failed to acquire lock on ", fullPath)
	tryLockTimes := 3
	for i := 0; i < tryLockTimes; i++ {
		cmd, stdout, err := subTryLockPath(fullPath)
		require.NoError(t, err)
		stderrBuf := new(strings.Builder)
		_, err = io.Copy(stderrBuf, stdout)
		require.NoError(t, err)
		require.Error(t, cmd.Wait())
		require.Contains(t, stderrBuf.String(), expectedErrMsg)
	}
	require.NoError(t, lockfile.UnlockAndDelete())

	cmd, _, err := subTryLockPath(fullPath)
	require.NoError(t, err)
	require.NoError(t, cmd.Wait())

	require.Len(t, stagingLockFiles, 0)
}
