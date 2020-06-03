package config

import (
	"strings"
	"testing"
)

const (
	foobar     = "foobar"
	nodev      = "nodev"
	trueString = "true"
	s100       = "100"
	s200       = "200"
)

func searchOptions(options []string, value string) bool {
	for _, s := range options {
		if strings.Contains(s, value) {
			return true
		}
	}
	return false
}

func TestAufsOptions(t *testing.T) {
	var (
		doptions []string
		options  OptionsConfig
	)
	doptions = GetGraphDriverOptions("aufs", options)
	if len(doptions) != 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	// Make sure legacy mountopt still works
	options = OptionsConfig{}
	options.MountOpt = foobar
	doptions = GetGraphDriverOptions("aufs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "mountopt=foobar") {
		t.Fatalf("Expected to find 'foobar' options, got %v", doptions)
	}

	// Make sure Aufs ignores other drivers mountpoints takes presedence
	options.Zfs.MountOpt = nodev
	doptions = GetGraphDriverOptions("aufs", options)
	if searchOptions(doptions, "mountopt=nodev") {
		t.Fatalf("Expected to find 'nodev' options, got %v", doptions)
	}

	// Make sure AufsMountOpt takes precedence
	options.Aufs.MountOpt = nodev
	doptions = GetGraphDriverOptions("aufs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "mountopt=nodev") {
		t.Fatalf("Expected to find 'nodev' options, got %v", doptions)
	}
}

func TestDeviceMapperOptions(t *testing.T) {
	var (
		doptions []string
		options  OptionsConfig
	)
	doptions = GetGraphDriverOptions("devicemapper", options)
	if len(doptions) != 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	// Make sure legacy mountopt still works
	options = OptionsConfig{}
	options.MountOpt = foobar
	doptions = GetGraphDriverOptions("devicemapper", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "mountopt=foobar") {
		t.Fatalf("Expected to find 'foobar' options, got %v", doptions)
	}

	// Make sure Devicemapper ignores other drivers mountpoints takes presedence
	options.Zfs.MountOpt = nodev
	doptions = GetGraphDriverOptions("devicemapper", options)
	if searchOptions(doptions, "mountopt=nodev") {
		t.Fatalf("Expected to find 'nodev' options, got %v", doptions)
	}

	// Make sure DevicemapperMountOpt takes precedence
	options.Thinpool.MountOpt = nodev
	doptions = GetGraphDriverOptions("devicemapper", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "mountopt=nodev") {
		t.Fatalf("Expected to find 'nodev' options, got %v", doptions)
	}

	options = OptionsConfig{}
	options.Thinpool.AutoExtendPercent = "50"
	doptions = GetGraphDriverOptions("devicemapper", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "50") {
		t.Fatalf("Expected to find '50' options, got %v", doptions)
	}
	options.Size = s200
	doptions = GetGraphDriverOptions("devicemapper", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, s200) {
		t.Fatalf("Expected to find size %q options, got %v", s200, doptions)
	}
	// Make sure Thinpool.Size takes precedence
	options.Thinpool.Size = s100
	doptions = GetGraphDriverOptions("devicemapper", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, s100) {
		t.Fatalf("Expected to find size %q, got %v", s100, doptions)
	}

}

func TestBtrfsOptions(t *testing.T) {
	var (
		doptions []string
		options  OptionsConfig
	)
	doptions = GetGraphDriverOptions("btrfs", options)
	if len(doptions) != 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	// Make sure legacy mountopt still works
	options = OptionsConfig{}
	options.Btrfs.MinSpace = s100
	doptions = GetGraphDriverOptions("btrfs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, s100) {
		t.Fatalf("Expected to find %q options, got %v", s100, doptions)
	}

	options = OptionsConfig{}
	options.Size = s200
	doptions = GetGraphDriverOptions("btrfs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, s200) {
		t.Fatalf("Expected to find size %q options, got %v", s200, doptions)
	}
	// Make sure Btrfs.Size takes precedence
	options.Btrfs.Size = s100
	doptions = GetGraphDriverOptions("btrfs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, s100) {
		t.Fatalf("Expected to find size %q, got %v", s100, doptions)
	}

}

func TestOverlayOptions(t *testing.T) {
	var (
		doptions []string
		options  OptionsConfig
	)
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) != 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	options.Vfs.IgnoreChownErrors = trueString
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) != 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	options.Overlay.IgnoreChownErrors = trueString
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 1 options, got %v", doptions)
	}
	options.Overlay.IgnoreChownErrors = "false"
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}

	// Make sure legacy IgnoreChownErrors still works
	options = OptionsConfig{}
	options.IgnoreChownErrors = trueString
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 1 options, got %v", doptions)
	}
	// Make sure legacy mountopt still works
	options = OptionsConfig{}
	options.MountOpt = foobar
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "mountopt=foobar") {
		t.Fatalf("Expected to find 'foobar' options, got %v", doptions)
	}

	// Make sure Overlay ignores other drivers mountpoints takes presedence
	options.Zfs.MountOpt = nodev
	doptions = GetGraphDriverOptions("overlay", options)
	if searchOptions(doptions, "mountopt=nodev") {
		t.Fatalf("Expected to find 'nodev' options, got %v", doptions)
	}

	// Make sure OverlayMountOpt takes precedence
	options.Overlay.MountOpt = nodev
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "mountopt=nodev") {
		t.Fatalf("Expected to find 'nodev' options, got %v", doptions)
	}

	// Make sure mount_program takes precedence
	options.MountProgram = "/usr/bin/root_overlay"
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "mount_program=/usr/bin/root_overlay") {
		t.Fatalf("Expected to find 'root_overlay' options, got %v", doptions)
	}
	options.Overlay.MountProgram = "/usr/bin/fuse_overlay"
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "mount_program=/usr/bin/fuse_overlay") {
		t.Fatalf("Expected to find 'fuse_overlay' options, got %v", doptions)
	}
	options.Overlay.SkipMountHome = "true"
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "skip_mount_home") {
		t.Fatalf("Expected to find 'skip_mount_home' options, got %v", doptions)
	}

	// Make sure legacy mountopt still works
	options = OptionsConfig{}
	options.SkipMountHome = "true"
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, "skip_mount_home") {
		t.Fatalf("Expected to find 'skip_mount_home' options, got %v", doptions)
	}

	options.Size = s200
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, s200) {
		t.Fatalf("Expected to find size %q options, got %v", s200, doptions)
	}
	// Make sure Overlay.Size takes precedence
	options.Overlay.Size = s100
	doptions = GetGraphDriverOptions("overlay", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, s100) {
		t.Fatalf("Expected to find size %q, got %v", s100, doptions)
	}

}

func TestVfsOptions(t *testing.T) {
	var (
		doptions []string
		options  OptionsConfig
	)
	doptions = GetGraphDriverOptions("vfs", options)
	if len(doptions) != 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	options.Overlay.IgnoreChownErrors = trueString
	doptions = GetGraphDriverOptions("vfs", options)
	if len(doptions) != 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	options.Vfs.IgnoreChownErrors = trueString
	doptions = GetGraphDriverOptions("vfs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 1 options, got %v", doptions)
	}
	// Make sure legacy IgnoreChownErrors still works
	options = OptionsConfig{}
	options.IgnoreChownErrors = trueString
	doptions = GetGraphDriverOptions("vfs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 1 options, got %v", doptions)
	}
}

func TestZfsOptions(t *testing.T) {
	var (
		doptions []string
		options  OptionsConfig
	)
	doptions = GetGraphDriverOptions("zfs", options)
	if len(doptions) != 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	// Make sure legacy mountopt still works
	options = OptionsConfig{}
	options.Zfs.Name = foobar
	doptions = GetGraphDriverOptions("zfs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, options.Zfs.Name) {
		t.Fatalf("Expected to find 'foobar' options, got %v", doptions)
	}
	// Make sure Zfs ignores other drivers mountpoints takes presedence
	options.Aufs.MountOpt = nodev
	doptions = GetGraphDriverOptions("zfs", options)
	if searchOptions(doptions, "mountopt=nodev") {
		t.Fatalf("Expected Not to find 'nodev' options, got %v", doptions)
	}

	options.Size = s200
	doptions = GetGraphDriverOptions("zfs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, s200) {
		t.Fatalf("Expected to find size %q options, got %v", s200, doptions)
	}
	// Make sure Zfs.Size takes precedence
	options.Zfs.Size = s100
	doptions = GetGraphDriverOptions("zfs", options)
	if len(doptions) == 0 {
		t.Fatalf("Expected 0 options, got %v", doptions)
	}
	if !searchOptions(doptions, s100) {
		t.Fatalf("Expected to find size %q, got %v", s100, doptions)
	}
}
