//go:build !linux || !composefs || !cgo
// +build !linux !composefs !cgo

package overlay

import (
	"fmt"
)

func composeFsSupported() bool {
	return false
}

func generateComposeFsBlob(toc []byte, destFile string) error {
	return fmt.Errorf("composefs is not supported")
}

func mountErofsBlob(blobFile, mountPoint string) error {
	return fmt.Errorf("composefs is not supported")
}

func enableVerityRecursive(path string) error {
	return fmt.Errorf("composefs is not supported")
}
