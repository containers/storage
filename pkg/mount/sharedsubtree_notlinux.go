// +build !linux

package mount

// MakePrivate ensures a mounted filesystem has the PRIVATE mount option enabled.
// See the supported options in flags.go for further reference.
func MakePrivate(mountPoint string) error {
	return nil
}

// MakeUnbindable ensures a mounted filesystem has the UNBINDABLE mount option
// enabled. See the supported options in flags.go for further reference.
func MakeUnbindable(mountPoint string) error {
	return ensureMountedAs(mountPoint, "unbindable")
}
