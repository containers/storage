package homedir

import (
	"errors"
	"os"
	"path/filepath"
)

// DataHome (deprecated)
func GetDataHome() (string, error) {
	return DataHome()
}

// DataHome returns XDG_DATA_HOME.
// DataHome returns $HOME/.local/share and nil error if XDG_DATA_HOME is not set.
//
// See also https://standards.freedesktop.org/basedir-spec/latest/ar01s03.html
func DataHome() (string, error) {
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return xdgDataHome, nil
	}
	home := Get()
	if home == "" {
		return "", errors.New("could not get either XDG_DATA_HOME or HOME")
	}
	return filepath.Join(home, ".local", "share"), nil
}

// GetCacheHome (deprecated)
func GetCacheHome() (string, error) {
	return CacheHome()
}

// CacheHome returns XDG_CACHE_HOME.
// CacheHome returns $HOME/.cache and nil error if XDG_CACHE_HOME is not set.
//
// See also https://standards.freedesktop.org/basedir-spec/latest/ar01s03.html
func CacheHome() (string, error) {
	if xdgCacheHome := os.Getenv("XDG_CACHE_HOME"); xdgCacheHome != "" {
		return xdgCacheHome, nil
	}
	home := Get()
	if home == "" {
		return "", errors.New("could not get either XDG_CACHE_HOME or HOME")
	}
	return filepath.Join(home, ".cache"), nil
}

// GetRuntimeDir (deprecated)
func GetRuntimeDir() (string, error) {
	return RuntimeDir()
}

// GetShortcutString (deprecated)
func GetShortcutString() string {
	return ShortcutString()
}
