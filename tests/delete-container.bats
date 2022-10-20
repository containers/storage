#!/usr/bin/env bats

load helpers

@test "delete-container" {
	# Create a layer.
	run storage --debug=false create-layer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	layer=$output

	# Create an image using that layer.
	run storage --debug=false create-image $layer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	image=${output%%	*}

	# Create an image using that layer.
	run storage --debug=false create-container $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	container=${output%%	*}

	# Check that the container can be found.
	storage exists -c $container

	# Use delete-container to delete it.
	storage delete-container $container

	# Check that the container is gone.
	run storage exists -c $container
	[ "$status" -ne 0 ]
}

@test "delete-container-with-immutable" {
	if [ "$OS" != "FreeBSD" ]; then
		skip "not supported on $OS"
	fi

	# Create a layer.
	run storage --debug=false create-layer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	layer=$output

	# Create an image using that layer.
	run storage --debug=false create-image $layer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	image=${output%%	*}

	# Create an image using that layer.
	run storage --debug=false create-container $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	container=${output%%	*}

	# Check that the container can be found.
	storage exists -c $container

	run storage --debug=false mount $container
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	containermount="$output"

	# Create a file and make it immutable
	createrandom "$containermount"/file1
	chflags schg "$containermount"/file1

	run storage --debug=false unmount $container
	[ "$status" -eq 0 ]
	[ "$output" != "" ]

	# Use delete-container to delete it.
	storage delete-container $container

	# Check that the container is gone.
	run storage exists -c $container
	[ "$status" -ne 0 ]
}
