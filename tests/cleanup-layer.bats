#!/usr/bin/env bats

load helpers

@test "cleanup-layer" {
	# Create a layer.
	run storage --debug=false create-layer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	sed -i -e 's/"id":/"flags":{"incomplete":true},"id":/g' ${TESTDIR}/root/${STORAGE_DRIVER}-layers/layers.json

	# Get a list of the layers, which should clean it up.
	run storage --debug=false layers
	[ "$status" -eq 0 ]
	echo "$output"
	[ "${#lines[*]}" -eq 0 ]
}
