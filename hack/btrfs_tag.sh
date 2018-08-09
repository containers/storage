#!/bin/bash
if test $(${GO:-go} env GOOS) != "linux" ; then
	exit 0
fi
cc -E - > /dev/null 2> /dev/null <<- EOF
#include <btrfs/ioctl.h>
EOF
if test $? -ne 0 ; then
	echo exclude_graphdriver_btrfs
else
	cc -E - > /dev/null 2> /dev/null <<- EOF
	#include <btrfs/version.h>
	EOF
	if test $? -ne 0 ; then
		echo btrfs_noversion
	fi
fi
