#!/usr/bin/env bats

load helpers

@test "applyjunk" {
	# Create and try to populate layers with... garbage.  It should be
	# rejected cleanly.
	for compressed in cat gzip bzip2 xz zstd ; do
		storage create-layer --id layer-${compressed}

		echo [[${compressed} /etc/os-release]]
		${compressed} < /etc/os-release > junkfile
		run storage apply-diff --file junkfile layer-${compressed}
		echo "$output"
		[[ "$status" -ne 0 ]]
		[[ "$output" =~ "invalid tar header" ]] || [[ "$output" =~ "unexpected EOF" ]]

		echo [[${compressed}]]
		echo "sorry, not even enough info for a tar header here" | ${compressed} > junkfile
		run storage apply-diff --file junkfile layer-${compressed}
		echo "$output"
		[[ "$status" -ne 0 ]]
		[[ "$output" =~ "unexpected EOF" ]]
	done
}
