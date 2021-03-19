#!/usr/bin/env bats

load helpers

@test "overlay-recreate" {
	case "$STORAGE_DRIVER" in
	overlay)
		;;
	*)
		skip "not applicable to driver $STORAGE_DRIVER"
		;;
	esac
	populate
	# behold my destructive power!
	rm -v ${TESTDIR}/root/overlay/l/*
	# we should be able to recover from that.
	storage mount "$lowerlayer"
	storage unmount "$lowerlayer"
	storage mount "$midlayer"
	storage unmount "$midlayer"
	storage mount "$upperlayer"
	storage unmount "$upperlayer"
	# okay, but how about this?
	rm -v ${TESTDIR}/root/overlay/*/link
	# yeah, we can handle that, too.
	storage mount "$lowerlayer"
	storage unmount "$lowerlayer"
	storage mount "$midlayer"
	storage unmount "$midlayer"
	storage mount "$upperlayer"
	storage unmount "$upperlayer"
	# okay, not bad, kid.
}
