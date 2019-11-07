#!/usr/bin/env bats

load helpers

@test "mount-image" {
	# Create and populate three interesting layers.
	populate

	# Create an image using to top layer.
	name=wonderful-image
	run storage --debug=false create-image --name $name $upperlayer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	image=${lines[0]}
	# Mount the layer.
	run storage --debug=false mount --ro $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	# Check if layer is mounted.
	run storage --debug=false mounted $image
	[ "$status" -eq 0 ]
	[ "$output" == "$image mounted" ]
	# Unmount the layer.
	run storage --debug=false unmount $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	# Make sure layer is not mounted.
	run storage --debug=false mounted $image
	[ "$status" -eq 0 ]
	[ "$output" == "" ]

	# Mount the image twice.
	run storage --debug=false mount $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	run storage --debug=false mount $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	# Check if image is mounted.
	run storage --debug=false mounted $image
	[ "$status" -eq 0 ]
	[ "$output" == "$image mounted" ]
	# Unmount the second image.
	run storage --debug=false unmount $image
	[ "$status" -eq 0 ]
	[ "$output" == "" ]
	# Check if layer is mounted.
	run storage --debug=false mounted $image
	[ "$status" -eq 0 ]
	[ "$output" == "$image mounted" ]
	# Unmount the image layer.
	run storage --debug=false unmount $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	# Make sure image is not mounted.
	run storage --debug=false mounted $image
	[ "$status" -eq 0 ]
	[ "$output" == "" ]

   	# Delete the image
	run storage delete-image $image
	[ "$status" -eq 0 ]
}

@test "container-on-mounted-image" {
    #Create and populate three intresting layers.
    populate

   	# Create an image using to top layer.
	name=wonderful-image
	run storage --debug=false create-image --name $name $upperlayer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	image=${lines[0]}

	# Create an image using to top layer.
	name=wonderful-image
	run storage --debug=false create-image --name $name $upperlayer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	image=${lines[0]}
	# Mount the image.
	run storage --debug=false mount --ro $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	# Check if image is mounted.
	run storage --debug=false mounted $image
	[ "$status" -eq 0 ]
	[ "$output" == "$image mounted" ]
   	# Create a container based on that image.
	run storage --debug=false create-container $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	container=${output%%	*}

    # Check that the container can be found.
	storage exists -c $container

	# Unmount the image.
	run storage --debug=false unmount $image
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	# Make sure image is not mounted.
	run storage --debug=false mounted $image
	[ "$status" -eq 0 ]
	[ "$output" == "" ]

   	# Use delete-container to delete it.
	storage delete-container $container

	# Check that the container is gone.
	run storage exists -c $container
	[ "$status" -ne 0 ]

   	# Delete the image
	run storage delete-image $image
	[ "$status" -eq 0 ]
}