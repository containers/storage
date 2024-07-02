#!/usr/bin/env bats

load helpers

function applyjunk_main() {
	# Create and try to populate layers with... garbage.  It should be
	# rejected cleanly.
	compressed="$1"

	storage create-layer --id layer-${compressed}

	echo [[${compressed} /etc/os-release]]
	if ! ${compressed} < /etc/os-release > ${TESTDIR}/junkfile ; then
		skip "error running ${compressed}"
	fi
	run storage apply-diff --file ${TESTDIR}/junkfile layer-${compressed}
	echo "$output"
	[[ "$status" -ne 0 ]]
	[[ "$output" =~ "invalid tar header" ]] || [[ "$output" =~ "unexpected EOF" ]]

	echo [[${compressed}]]
	echo "sorry, not even enough info for a tar header here" | ${compressed} > ${TESTDIR}/junkfile
	run storage apply-diff --file ${TESTDIR}/junkfile layer-${compressed}
	echo "$output"
	[[ "$status" -ne 0 ]]
	[[ "$output" =~ "unexpected EOF" ]]
}

@test "applyjunk-uncompressed" {
	applyjunk_main cat
}

@test "applyjunk-gzip" {
	applyjunk_main gzip
}

@test "applyjunk-bzip2" {
	applyjunk_main bzip2
}

@test "applyjunk-xz" {
	applyjunk_main xz
}

@test "applyjunk-zstd" {
	applyjunk_main zstd
}
