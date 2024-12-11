package path

import (
	"path/filepath"
)

// CleanAbsPath removes any ".." and "." from the path
// and ensures it starts with a "/".  If the path refers to the root
// directory, it returns "/".
func CleanAbsPath(path string) string {
	return filepath.Clean("/" + path)
}
