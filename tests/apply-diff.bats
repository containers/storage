#!/usr/bin/env bats

load helpers

@test "applydiff" {
	# The checkdiffs function needs "tar".
	if test -z "$(which tar 2> /dev/null)" ; then
		skip "need tar"
	fi

	# Create and populate three interesting layers.
	populate

	# Extract the layers.
	storage diff -u -f $TESTDIR/lower.tar $lowerlayer
	storage diff -c -f $TESTDIR/middle.tar $midlayer
	storage diff -u -f $TESTDIR/upper.tar $upperlayer

	# Delete the layers.
	storage delete-layer $upperlayer
	storage delete-layer $midlayer
	storage delete-layer $lowerlayer

	# Create new layers and populate them using the layer diffs.
	run storage --debug=false create-layer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowerlayer="$output"
	storage applydiff -f $TESTDIR/lower.tar "$lowerlayer"

	run storage --debug=false create-layer "$lowerlayer"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	midlayer="$output"
	storage applydiff -f $TESTDIR/middle.tar "$midlayer"

	run storage --debug=false create-layer "$midlayer"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	upperlayer="$output"
	storage applydiff -f $TESTDIR/upper.tar "$upperlayer"

	# The contents of these new layers should match what the old ones had.
	checkchanges
	checkdiffs
}

@test "apply-implicitdir-diff" {
	# We need "tar" to build layer diffs.
	if test -z "$(which tar 2> /dev/null)" ; then
		skip "need tar"
	fi

	# Create one layer diff, then another that includes added/modified
	# items but _not_ the directories that contain them.
	pushd $TESTDIR
	mkdir subdirectory1
	chmod 0700 subdirectory1
	mkdir subdirectory2
	chmod 0750 subdirectory2
	tar cvf lower.tar subdirectory1 subdirectory2
	touch subdirectory1/testfile1 subdirectory2/testfile2
	tar cvf middle.tar subdirectory1/testfile1 subdirectory2/testfile2
	popd

	# Create layers and populate them using the diffs.
	run storage --debug=false create-layer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowerlayer="$output"
	storage applydiff -f "$TESTDIR"/lower.tar "$lowerlayer"

	run storage --debug=false create-layer "$lowerlayer"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	middlelayer="$output"
	storage applydiff -f "$TESTDIR"/middle.tar "$middlelayer"

	run storage --debug=false create-layer "$middlelayer"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	upperlayer="$output"

	run storage --debug=false mount "$upperlayer"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	mountpoint="$output"

	run stat -c %a "$TESTDIR"/subdirectory1
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	expected="$output"
	echo subdirectory1 should have mode $expected

	run stat -c %a "$mountpoint"/subdirectory1
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	actual="$output"
	echo subdirectory1 has mode $actual
	[ "$actual" = "$expected" ]

	run stat -c %a "$TESTDIR"/subdirectory2
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	expected="$output"
	echo subdirectory2 should have mode $expected

	run stat -c %a "$mountpoint"/subdirectory2
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	actual="$output"
	echo subdirectory2 has mode $actual
	[ "$actual" = "$expected" ]
}
