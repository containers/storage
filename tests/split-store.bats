#!/usr/bin/env bats

load helpers

@test "split-store" {
	# Create and populate three interesting layers.
	populate

	# Create an image using to top layer.
	name=wonderful-image
	run mkdir -p ${TESTDIR}/imagestore
	run mkdir -p ${TESTDIR}/emptyimagestore
	run storage --graph ${TESTDIR}/graph/ --image-store ${TESTDIR}/imagestore/ --run ${TESTDIR}/runroot/ --debug=false create-image --name $name
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	image=${lines[0]}

	# Add a couple of big data items.
	createrandom ${TESTDIR}/random1
	createrandom ${TESTDIR}/random2
	storage --graph ${TESTDIR}/graph/ --image-store ${TESTDIR}/imagestore/ --run ${TESTDIR}/runroot/ set-image-data -f ${TESTDIR}/random1 $image random1
	storage --graph ${TESTDIR}/graph/ --image-store ${TESTDIR}/imagestore/ --run ${TESTDIR}/runroot/ set-image-data -f ${TESTDIR}/random2 $image random2

	# Get information about the image, and make sure the ID, name, and data names were preserved.
	run storage --graph ${TESTDIR}/graph/ --image-store ${TESTDIR}/imagestore/ --run ${TESTDIR}/runroot/ image $image
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" =~ "ID: $image" ]]
	[[ "$output" =~ "Name: $name" ]]
	[[ "$output" =~ "Data: random1" ]]
	[[ "$output" =~ "Data: random2" ]]

	# shutdown store
	run storage --graph ${TESTDIR}/graph/ --image-store ${TESTDIR}/imagestore/ --run ${TESTDIR}/runroot/ shutdown

	# Similar data must not be shown when image-store is switched to empty store
	run storage --graph ${TESTDIR}/graph/ --image-store ${TESTDIR}/emptyimagestore/ --run ${TESTDIR}/runroot/ image $image
	echo "$output"
	[[ "$output" != "ID: $image" ]]
	[[ "$output" != "Name: $name" ]]
	[[ "$output" != "Data: random1" ]]
	[[ "$output" != "Data: random2" ]]

	# shutdown store
	run storage --graph ${TESTDIR}/graph/ --image-store ${TESTDIR}/emptyimagestore/ --run ${TESTDIR}/runroot/ shutdown
}

@test "split-store - use graphRoot as an additional store by default" {
	case "$STORAGE_DRIVER" in
	overlay*)
		;;
	*)
		skip "additional store not supported by driver $STORAGE_DRIVER"
		;;
	esac
	# Create and populate three interesting layers.
	populate

	# Create an image using to top layer.
	name=wonderful-image
	run mkdir -p ${TESTDIR}/imagestore
	run storage --graph ${TESTDIR}/graph --debug=false create-image --name $name
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	image=${lines[0]}

	# Add a couple of big data items.
	createrandom ${TESTDIR}/random1
	createrandom ${TESTDIR}/random2
	storage --graph ${TESTDIR}/graph set-image-data -f ${TESTDIR}/random1 $image random1
	storage --graph ${TESTDIR}/graph set-image-data -f ${TESTDIR}/random2 $image random2

	# Get information about the image, and make sure the ID, name, and data names were preserved.
	run storage --graph ${TESTDIR}/graph image $image
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" =~ "ID: $image" ]]
	[[ "$output" =~ "Name: $name" ]]
	[[ "$output" =~ "Data: random1" ]]
	[[ "$output" =~ "Data: random2" ]]

	# shutdown store
	run storage --graph ${TESTDIR}/graph shutdown

	# Similar data must not be shown when image-store is switched to empty store
	run storage --graph ${TESTDIR}/graph --image-store ${TESTDIR}/imagestore/ --run ${TESTDIR}/runroot/ image $image
	echo "$output"
	[[ "$output" =~ "ID: $image" ]]
	[[ "$output" =~ "Name: $name" ]]
	[[ "$output" =~ "Data: random1" ]]
	[[ "$output" =~ "Data: random2" ]]
	# Since this image is being read from the readonly graph root
	# so it must show that
	[[ "$output" =~ "Read Only: true" ]]

	# shutdown store
	run storage --graph ${TESTDIR}/graph --image-store ${TESTDIR}/imagestore/ --run ${TESTDIR}/runroot/ shutdown
}
