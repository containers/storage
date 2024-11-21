#!/usr/bin/env bats

load helpers

@test "dedup" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support dedup"
		;;
	esac

	if test -z "$(which tar 2> /dev/null)" ; then
		skip "need tar"
	fi

	if test -z "$(which jq 2> /dev/null)" ; then
		skip "need jq"
	fi

	echo some content > $TESTDIR/from
	# Skip the test if the underlying file system does not support reflinks.
	if ! cp --reflink=always $TESTDIR/from $TESTDIR/to; then
		skip "need reflink support"
	fi

	populate

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
	storage applydiff -f $TESTDIR/lower.tar "$upperlayer"
	storage applydiff -f $TESTDIR/upper.tar "$upperlayer"

	for layer in $lowerlayer $midlayer $upperlayer; do
		run storage --debug=false create-image $layer
		[ "$status" -eq 0 ]
	done

	run storage --debug=false dedup -j
	[ "$status" -eq 0 ]
	deduped=$(jq -r .Deduped <<< $output)
	[[ $deduped -gt 0 ]]

	for METHOD in size crc sha256; do
		# Test that it always returns the same value with any hash-method.
		for i in $(seq 10); do
			run storage --debug=false dedup -j --hash-method=$METHOD
			[ "$status" -eq 0 ]
			actual=$(jq -r .Deduped <<< $output)
			[[ $deduped = $actual ]]
	        done
        done
}
