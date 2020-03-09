// +build !linux,!freebsd,!solaris freebsd,!cgo solaris,!cgo

package mount

func mount(device, target, mType string, flag uintptr, data string) error {
	panic("Not implemented")
}
