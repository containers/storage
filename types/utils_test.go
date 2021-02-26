package types

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/storage/pkg/homedir"
	"github.com/containers/storage/pkg/system"
	"gotest.tools/assert"
)

type homeRuntimeData struct {
	dir string
	err error
}

type rootlessRuntimeDirEnvironmentTest struct {
	homeRuntime     homeRuntimeData
	procCommandFile string
	runUserDir      string
	tmpPerUserDir   string
	homeDir         string
	result          string
}

func (env rootlessRuntimeDirEnvironmentTest) getProcCommandFile() string {
	return env.procCommandFile
}
func (env rootlessRuntimeDirEnvironmentTest) getRunUserDir() string {
	return env.runUserDir
}
func (env rootlessRuntimeDirEnvironmentTest) getTmpPerUserDir() string {
	return env.tmpPerUserDir
}
func (env rootlessRuntimeDirEnvironmentTest) homeDirGetRuntimeDir() (string, error) {
	return env.homeRuntime.dir, env.homeRuntime.err
}
func (env rootlessRuntimeDirEnvironmentTest) systemLstat(path string) (*system.StatT, error) {
	return system.Lstat(path)
}
func (env rootlessRuntimeDirEnvironmentTest) homedirGet() string {
	return env.homeDir
}

func TestRootlessRuntimeDir(t *testing.T) {
	testDir, err := ioutil.TempDir("", "rootless-runtime-dir-test")
	assert.NilError(t, err)
	defer os.Remove(testDir)

	homeRuntimeDir := filepath.Join(testDir, "home-rundir")
	err = os.Mkdir(homeRuntimeDir, 0700)
	assert.NilError(t, err)

	homeRuntimeDisabled := homeRuntimeData{err: errors.New("homedirGetRuntimeDir is disabled")}

	systemdCommandFile := filepath.Join(testDir, "systemd-command")
	err = ioutil.WriteFile(systemdCommandFile, []byte("systemd"), 0644)
	assert.NilError(t, err)

	initCommandFile := filepath.Join(testDir, "init-command")
	err = ioutil.WriteFile(initCommandFile, []byte("init"), 0644)
	assert.NilError(t, err)

	dirForOwner := filepath.Join(testDir, "dir-for-owner")
	err = os.Mkdir(dirForOwner, 0700)
	assert.NilError(t, err)

	dirForAll := filepath.Join(testDir, "dir-for-all")
	err = os.Mkdir(dirForAll, 0777)
	assert.NilError(t, err)

	dirToBeCreated := filepath.Join(testDir, "dir-to-be-created")

	envs := []rootlessRuntimeDirEnvironmentTest{
		{
			homeRuntime: homeRuntimeData{dir: homeRuntimeDir},
			result:      homeRuntimeDir,
		},

		// Reading proc command file fails
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: "",
			runUserDir:      dirForOwner,
			result:          dirForOwner,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: "",
			runUserDir:      "", // Accessing run user dir fails
			tmpPerUserDir:   dirForOwner,
			result:          dirForOwner,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: "",
			runUserDir:      dirForAll,
			tmpPerUserDir:   dirForOwner,
			result:          dirForOwner,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: "",
			runUserDir:      "", // Accessing run user dir fails
			tmpPerUserDir:   dirToBeCreated,
			result:          dirToBeCreated,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: "",
			runUserDir:      "", // Accessing run user dir fails
			tmpPerUserDir:   "", // Accessing tmp per user dir fails
			homeDir:         dirForOwner,
			result:          filepath.Join(dirForOwner, "rundir"),
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: "",
			runUserDir:      "", // Accessing run user dir fails
			tmpPerUserDir:   dirForAll,
			homeDir:         dirForOwner,
			result:          filepath.Join(dirForOwner, "rundir"),
		},

		// systemd
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: systemdCommandFile,
			runUserDir:      dirForOwner,
			result:          dirForOwner,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: systemdCommandFile,
			runUserDir:      "", // Accessing run user dir fails
			tmpPerUserDir:   dirForOwner,
			result:          dirForOwner,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: systemdCommandFile,
			runUserDir:      dirForAll,
			tmpPerUserDir:   dirForOwner,
			result:          dirForOwner,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: systemdCommandFile,
			runUserDir:      "", // Accessing run user dir fails
			tmpPerUserDir:   dirToBeCreated,
			result:          dirToBeCreated,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: systemdCommandFile,
			runUserDir:      "", // Accessing run user dir fails
			tmpPerUserDir:   "", // Accessing tmp per user dir fails
			homeDir:         dirForOwner,
			result:          filepath.Join(dirForOwner, "rundir"),
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: systemdCommandFile,
			runUserDir:      "", // Accessing run user dir fails
			tmpPerUserDir:   dirForAll,
			homeDir:         dirForOwner,
			result:          filepath.Join(dirForOwner, "rundir"),
		},

		// init
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: initCommandFile,
			tmpPerUserDir:   dirForOwner,
			result:          dirForOwner,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: initCommandFile,
			tmpPerUserDir:   dirToBeCreated,
			result:          dirToBeCreated,
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: initCommandFile,
			tmpPerUserDir:   "", // Accessing tmp per user dir fails
			homeDir:         dirForOwner,
			result:          filepath.Join(dirForOwner, "rundir"),
		},
		{
			homeRuntime:     homeRuntimeDisabled,
			procCommandFile: initCommandFile,
			tmpPerUserDir:   dirForAll,
			homeDir:         dirForOwner,
			result:          filepath.Join(dirForOwner, "rundir"),
		},
	}

	for i, env := range envs {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			os.Remove(dirToBeCreated)

			resultDir, err := getRootlessRuntimeDirIsolated(env)
			assert.NilError(t, err)
			assert.Assert(t, resultDir == env.result)
		})
	}
}

type rootlessRuntimeDirEnvironmentRace struct {
	procCommandFile string
	tmpPerUserDir   string
}

func (env rootlessRuntimeDirEnvironmentRace) getProcCommandFile() string {
	return env.procCommandFile
}
func (rootlessRuntimeDirEnvironmentRace) getRunUserDir() string {
	return ""
}
func (env rootlessRuntimeDirEnvironmentRace) getTmpPerUserDir() string {
	return env.tmpPerUserDir
}
func (rootlessRuntimeDirEnvironmentRace) homeDirGetRuntimeDir() (string, error) {
	return "", errors.New("homedirGetRuntimeDir is disabled")
}
func (env rootlessRuntimeDirEnvironmentRace) systemLstat(path string) (*system.StatT, error) {
	if path == env.tmpPerUserDir {
		st, err := system.Lstat(path)
		// We can simulate that race directory was created immediately after system.Lstat call.
		if err := os.Mkdir(path, 0700); err != nil {
			return nil, err
		}
		return st, err
	}
	return system.Lstat(path)
}
func (rootlessRuntimeDirEnvironmentRace) homedirGet() string {
	return homedir.Get()
}

func TestRootlessRuntimeDirRace(t *testing.T) {
	raceDir, err := ioutil.TempDir("", "rootless-runtime-dir-race-test")
	assert.NilError(t, err)
	defer os.Remove(raceDir)

	procCommandFile := filepath.Join(raceDir, "command")
	err = ioutil.WriteFile(procCommandFile, []byte("init"), 0644)
	assert.NilError(t, err)

	tmpPerUserDir := filepath.Join(raceDir, "tmp")

	resultDir, err := getRootlessRuntimeDirIsolated(rootlessRuntimeDirEnvironmentRace{
		procCommandFile,
		tmpPerUserDir,
	})
	assert.NilError(t, err)
	assert.Assert(t, resultDir != tmpPerUserDir, "Rootless runtime dir shouldn't follow race dir.")
}

func TestDefaultStoreOpts(t *testing.T) {
	storageOpts, err := defaultStoreOptionsIsolated(true, 1000, "./storage_test.conf")

	expectedPath := filepath.Join(os.Getenv("HOME"), "1000", "containers/storage")

	assert.NilError(t, err)
	assert.Equal(t, storageOpts.RunRoot, expectedPath)
	assert.Equal(t, storageOpts.GraphRoot, expectedPath)
	assert.Equal(t, storageOpts.RootlessStoragePath, expectedPath)
}
