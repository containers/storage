package homedir

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// GetConfigHome returns XDG_CONFIG_HOME.
// GetConfigHome returns $HOME/.config and nil error if XDG_CONFIG_HOME is not set.
//
// See also https://standards.freedesktop.org/basedir-spec/latest/ar01s03.html
func GetConfigHome() (string, error) {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return xdgConfigHome, nil
	}
	home := Get()
	if home == "" {
		return "", errors.New("could not get either XDG_CONFIG_HOME or HOME")
	}
	return filepath.Join(home, ".config"), nil
}

// GetDataHome returns XDG_DATA_HOME.
// GetDataHome returns $HOME/.local/share and nil error if XDG_DATA_HOME is not set.
//
// See also https://standards.freedesktop.org/basedir-spec/latest/ar01s03.html
func GetDataHome() (string, error) {
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return xdgDataHome, nil
	}
	home := Get()
	if home == "" {
		return "", errors.New("could not get either XDG_DATA_HOME or HOME")
	}
	return filepath.Join(home, ".local", "share"), nil

}

// GetDataHomeByUID finds the home directory for the given uid, only works with sudo permissions
func GetDataHomeByUID(uid int) (string, error) {
	var home string
	u, err := user.LookupId(fmt.Sprint(uid))
	if err != nil {
		return "", err
	}
	cmd := exec.Command("sudo", "-Hiu", u.Username, "env")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	env := strings.Split(string(out), "\n")
	for _, val := range env {
		if strings.Contains(val, "XDG_DATA_HOME") || strings.Contains(val, "HOME") {
			home = strings.Split(val, "=")[1]
		}
	}
	if len(home) == 0 {
		return "", errors.New("called function as root, could not find given user's home directory")
	}
	return home, nil
}

// GetCacheHome returns XDG_CACHE_HOME.
// GetCacheHome returns $HOME/.cache and nil error if XDG_CACHE_HOME is not set.
//
// See also https://standards.freedesktop.org/basedir-spec/latest/ar01s03.html
func GetCacheHome() (string, error) {
	if xdgCacheHome := os.Getenv("XDG_CACHE_HOME"); xdgCacheHome != "" {
		return xdgCacheHome, nil
	}
	home := Get()
	if home == "" {
		return "", errors.New("could not get either XDG_CACHE_HOME or HOME")
	}
	return filepath.Join(home, ".cache"), nil
}
