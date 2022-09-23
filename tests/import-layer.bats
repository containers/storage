#!/usr/bin/env bats

load helpers

@test "import-layer" {
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

	# Import new layers using the layer diffs.
	run storage --debug=false import-layer -f $TESTDIR/lower.tar
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowerlayer="$output"

	run storage --debug=false import-layer -f $TESTDIR/middle.tar "$lowerlayer"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	midlayer="$output"

	run storage --debug=false import-layer -f $TESTDIR/upper.tar "$midlayer"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	upperlayer="$output"

	# The contents of these new layers should match what the old ones had.
	checkchanges
	checkdiffs
}

set_immutable() {
	chflags schg $1
}

reset_immutable() {
	chflags noschg $1
}

is_immutable() {
	local flags=$(stat -f %#Xf $1)
	[ "$((($flags & 0x20000) == 0x20000))" -ne 0 ]
}

@test "import-layer-with-immutable" {
	if [ "$OS" != "FreeBSD" ]; then
		skip "not supported on $OS"
	fi

	# Create a layer with a directory containing two files, both
	# immutable. The directory is also set as immutablr.
	run storage --debug=false create-layer
	echo $output
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowerlayer="$output"
	run storage --debug=false mount $lowerlayer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	local m="$output"
	mkdir $m/dir
	createrandom $m/dir/layer1file1
	createrandom $m/dir/layer1file2
	set_immutable $m/dir/layer1file1
	set_immutable $m/dir/layer1file2
	set_immutable $m/dir
	storage unmount $lowerlayer

	# Create a second layer which deletes one file and removes immutable from the other
	run storage --debug=false create-layer "$lowerlayer"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	upperlayer="$output"
	run storage --debug=false mount $upperlayer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	m="$output"
	reset_immutable $m/dir
	reset_immutable $m/dir/layer1file1
	rm $m/dir/layer1file1
	reset_immutable $m/dir/layer1file2
	set_immutable $m/dir
	storage unmount $upperlayer

	# Extract the layers.
	storage diff -u -f $TESTDIR/lower.tar $lowerlayer
	storage diff -u -f $TESTDIR/upper.tar $upperlayer

	# Delete the layers.
	storage delete-layer $upperlayer
	storage delete-layer $lowerlayer

	# Import new layers using the layer diffs.
	run storage --debug=false import-layer -f $TESTDIR/lower.tar
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowerlayer="$output"

	run storage --debug=false import-layer -f $TESTDIR/upper.tar "$lowerlayer"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	upperlayer="$output"

	# Verify layer contents
	run storage --debug=false mount $lowerlayer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	m="$output"
	is_immutable $m/dir/layer1file1
	is_immutable $m/dir/layer1file2
	storage unmount $lowerlayer

	run storage --debug=false mount $upperlayer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	m="$output"
	[ ! -f $m/dir/layer1file1 ]
	! is_immutable $m/dir/layer1file2
	storage unmount $upperlayer

	storage delete-layer $upperlayer
	storage delete-layer $lowerlayer
}
