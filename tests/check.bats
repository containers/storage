#!/usr/bin/env bats

load helpers

# Check that the storage driver doesn't have any layers that we don't know
# about, and would therefore never be able to clean up, i.e., that we can
# spot them.
@test "check-unmanaged-layers" {
	run storage --debug=false storage-layers
	echo storage-layers: "$output"
	if [ $status -eq 1 -a "$output" == "driver not supported" ] ; then
		skip "driver $STORAGE_DRIVER does not support ListLayers()"
	fi

	run storage --debug=false create-storage-layer
	echo create-storage-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	run storage create-storage-layer "$layer"
	echo create-storage-layer: "$output"
	[[ $status -eq 0 ]]

	run storage create-storage-layer
	echo create-storage-layer: "$output"
	[[ $status -eq 0 ]]

	run storage check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "layer ${layer}: layer in lower level storage driver not accounted for" ]]

	run storage --debug=false storage-layers
	echo storage-layers: "$output"
	[[ $status -eq 0 ]]
	[[ $output == "" ]]
}

# Check that nothing happens when the storage driver had layers that we don't
# know about in a read-only store.  It's not as if we can do anything about
# them.
@test "check-unmanaged-ro-layers" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Put a couple of unmanaged layers in the read-only location.
	mkdir ${TESTDIR}/{ro-root,ro-runroot}

	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-storage-layer
	echo create-storage-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-storage-layer "$layer"
	echo create-storage-layer: "$output"
	[[ $status -eq 0 ]]
	otherlayer="$output"

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Put an image in the read-write location.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-layer
	echo create-storage-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-image "$layer"
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Check that we don't complain about unmanaged layers in the read-only location.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check
	echo "check:" "$output"
	[[ $status -eq 0 ]]
	[[ $output == "" ]]
}

# Check that the storage driver doesn't have any layers that we don't know
# about in a read-write store, and would therefore never be able to clean up,
# i.e., that we can spot them.
@test "check-unmanaged-rw-layers" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Put an image in the read-only location.
	mkdir ${TESTDIR}/{ro-root,ro-runroot}

	run storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rolayer=$output

	run storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-image "$rolayer"
	echo create-image: "$output"
	[[ $status -eq 0 ]]

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Put a couple of unmanaged layers in the read-write location.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-storage-layer
	echo create-storage-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-storage-layer "$layer"
	echo create-storage-layer: "$output"
	[[ $status -eq 0 ]]
	otherlayer=$output

	# Check that we find the unmanaged layers in the read-write location and remove them.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "layer ${layer}: layer in lower level storage driver not accounted for" ]]
	[[ $output =~ "layer ${otherlayer}: layer in lower level storage driver not accounted for" ]]

	# So now there shouldn't be any layers at all if we're just looking at the read-write location.
	run storage --debug=false storage-layers
	echo storage-layers: "$output"
	[[ $status -eq 0 ]]
	[[ $output == "" ]]

	# But there should still be that managed layer we put in the read-only location.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root layers -q
	echo storage-layers: "$output"
	[[ $status -eq 0 ]]
	[[ $output == "$rolayer" ]]
}

# Check that we can detect layers that aren't part of an image or a container.
@test "check-unused-layers" {
	run storage --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	run storage --debug=false create-layer $output
	[[ $status -eq 0 ]]
	layer=$output

	# By default, an unreferenced layer must have reached some minimum age
	# in order for us to think it's been forgotten.
	run storage --debug=false check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output == "" ]]

	# But if we set that minimum age to 0, we should clean it up.
	run storage check -r -m 0
	echo "check -r -m 0:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "layer ${layer}: layer not referenced by any images or containers" ]]

	# After the cleanup, there shouldn't be anything left.
	run storage --debug=false layers
	echo layers: "$output"
	[[ $status -eq 0 ]]
	[[ $output == "" ]]
}

# Check that we don't complain about layers in read-only storage that aren't
# part of an image or a container, since we can't do anything about them
# anyway.
@test "check-unused-ro-layers" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Put a couple of unreferenced layers in the read-only location.
	mkdir ${TESTDIR}/{ro-root,ro-runroot}
	run storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rolayer="$output"
	run storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-layer "$rolayer"
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	otherlayer="$output"

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Put an image in the read-write location.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-image "$layer"
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Check for errors.  We shouldn't be warning about the unreferenced read-only layers.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -m 0
	echo "check -m 0:" "$output"
	[[ $status -eq 0 ]]
	[[ $output == "" ]]
}

# Check that we can detect layers in read-write storage that aren't part of an
# image or a container.
@test "check-unused-rw-layers" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Put an image in the read-only location.
	mkdir ${TESTDIR}/{ro-root,ro-runroot}
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rolayer="$output"
	storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-image "$rolayer"

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Put some unreferenced layers in the read-write location.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-layer "$layer"
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	otherlayer=$output

	# By default, an unreferenced layer must have reached some minimum age
	# in order for us to think it's been forgotten.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output == "" ]]

	# Now check for errors and repair them.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r -m 0
	echo "check -r -m 0:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "layer $layer: layer not referenced by any images or containers" ]]
	[[ $output =~ "layer $otherlayer: layer not referenced by any images or containers" ]]

	# The read-only layer should be the only one present.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root layers -q
	echo layers: "$output"
	[[ $status -eq 0 ]]
	[[ $output == "$rolayer" ]]
}

# Check that we can detect when the contents of a layer's files, at least the
# ones that we'd need to read in order to reconstruct the diff, have been
# altered.
@test "check-layer-content-digest" {
	# This test needs "tar".
	if test -z "$(which tar 2> /dev/null)" ; then
		skip "need tar"
	fi

	# Create a layer.
	run storage --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Set contents of the layer.
	createrandom ${TESTDIR}/datafile1
	createrandom ${TESTDIR}/datafile2
	(cd ${TESTDIR}; tar cf - datafile1 datafile2) > ${TESTDIR}/diff
	storage apply-diff -f ${TESTDIR}/diff $layer

	# Mark that layer as part of an image.
	run storage --debug=false create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Put something in the layer that wasn't part of the diff.
	createrandom ${TESTDIR}/datafile3
	storage copy ${TESTDIR}/datafile3 ${layer}:/datafile1

	# Now check if the diff can be reproduced correctly.
	run storage check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "layer ${layer}: layer content incorrect digest" ]] || [[ $output =~ "layer ${layer}: file integrity checksum failed" ]]

	# Having removed the layer, there should be no traces left.
	run storage --debug=false images
	echo images: "$output"
	[[ $status -eq 0 ]]
	[[ $output == "" ]]

	# Should look empty now
	run storage --debug=false layers
	echo layers: "$output"
	[[ $status -eq 0 ]]
	[[ $output == "" ]]
}

# Check that we can detect when the contents of a read-only layer's files, at
# least the ones that we'd need to read in order to reconstruct the diff, have
# been altered.
@test "check-ro-layer-content-digest" {
	# This test needs "tar".
	if test -z "$(which tar 2> /dev/null)" ; then
		skip "need tar"
	fi

	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Put a layer record in the read-only location.
	mkdir ${TESTDIR}/{ro-root,ro-runroot}
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rolayer="$output"

	# Set up that layer's contents.
	createrandom ${TESTDIR}/datafile1
	createrandom ${TESTDIR}/datafile2
	(cd ${TESTDIR}; tar cf - datafile1 datafile2) > ${TESTDIR}/diff
	storage apply-diff --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot -f ${TESTDIR}/diff $rolayer

	# Mess with that layer's contents.
	createrandom ${TESTDIR}/datafile3
	storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot copy ${TESTDIR}/datafile3 ${rolayer}:/datafile1

	# Create an image record that uses that layer.
	run storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot --debug=false create-image "$rolayer"
	echo create-image: "$output"
	[[ $status -eq 0 ]]

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Create a read-write layer.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rwlayer=$output

	# Create a read-write image.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-image $rwlayer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Check that we notice the added file, even if we can't fix it.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r
	echo "check -r:" "$output"
	[[ $status -ne 0 ]] # couldn't fix read-only layers
	[[ $output =~ "layer ${rolayer}: layer content incorrect digest" ]] || [[ $output =~ "layer ${rolayer}: file integrity checksum failed" ]]

	# A check of just the read-write storage shouldn't turn up anything.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect when a layer has had content added.  Due to the
# way diff reconstructs diffs from layers, items which weren't in the original
# diff won't be noticed if the check consists of only extracting the diff.
@test "check-layer-content-modified" {
	# This test needs "tar".
	if test -z "$(which tar 2> /dev/null)" ; then
		skip "need tar"
	fi

	# Create the layer record.
	run storage --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Set up the layer's contents.
	createrandom ${TESTDIR}/datafile1
	createrandom ${TESTDIR}/datafile2
	(cd ${TESTDIR}; tar cf - datafile1 datafile2) > ${TESTDIR}/diff
	storage apply-diff -f ${TESTDIR}/diff $layer

	# Add some contents to the layer.
	createrandom ${TESTDIR}/datafile3
	storage copy ${TESTDIR}/datafile3 ${layer}:/datafile3

	# Create the image record.
	run storage --debug=false create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Check if we can detect that file being added.
	run storage check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "layer ${layer}: +/datafile3, layer content modified" ]]

	# Should be all clear now.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect when one of an image's layers is gone, or at least
# doesn't correspond to one that we know of.
@test "check-image-layer-missing" {
	# Create the layer record.
	run storage --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the image record.
	run storage --debug=false create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]
	image="$output"

	# Delete the layer with no safety checking.
	run storage --debug=false delete $layer
	echo delete layer: "$output"
	[[ $status -eq 0 ]]

	# Check that we know to flag the image as damaged, and fix it.
	run storage check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "image ${image}: layer ${layer}: image layer is missing" ]]

	# Check that we no longer think there's damage.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect when one of an image's layers is gone, or at least
# doesn't correspond to one that we know of.
@test "check-ro-image-layer-missing" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Create the read-only layer record.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the read-only image record.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]
	image="$output"

	# Delete the layer with no safety checking.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot delete $layer
	echo delete layer: "$output"
	[[ $status -eq 0 ]]

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Create a read-write layer.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rwlayer=$output

	# Create a read-write image.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-image $rwlayer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Check that we know to flag the image as damaged, even if we can't fix it.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r
	echo "check -r:" "$output"
	[[ $status -ne 0 ]] # we can't fix it
	[[ $output =~ "image ${image}: layer ${layer}: image layer is missing" ]]

	# Check that we no longer think there's damage if we just look at read-write content.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect when a container's base image is gone, or at least
# doesn't correspond to one that we know of.
@test "check-container-image-missing" {
	# Create the layer.
	run storage --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the image that uses that layer.
	run storage --debug=false create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]
	image=$output

	# Create a container based on that image.
	run storage --debug=false create-container $image
	echo create-container: "$output"
	[[ $status -eq 0 ]]
	container=$output

	# Delete the image with no safety checks.
	run storage --debug=false delete $image
	echo delete image: "$output"
	[[ $status -eq 0 ]]

	# Check and repair.  Repair is okay with deleting images because they
	# can be rebuilt or re-pulled.
	run storage check -r
	echo "check -r:" "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "container ${container}:" ]]
	[[ $output =~ "image ${image}: image missing" ]]

	# didn't get rid of the container, though!

	run storage check
	echo check: "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "container ${container}:" ]]
	[[ $output =~ "image ${image}: image missing" ]]

	# Repair, but now we're okay with getting rid of containers.
	run storage check -r -f
	echo "check -r -f:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "container ${container}:" ]]
	[[ $output =~ "image ${image}: image missing" ]]

	# Should be all clear now.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect when a container's base image is deleted in a
# read-only store, or at least doesn't correspond to one that we know of.
@test "check-container-ro-image-missing" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Create the read-only layer.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the read-only image that uses that layer.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]
	image=$output

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Create a container based on that image.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-container $image
	echo create-container: "$output"
	[[ $status -eq 0 ]]
	container=$output
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root container ${container}
	echo container: "$output"
	[[ $status -eq 0 ]]
	clayer=$(grep ^Layer: <<< ${output})
	clayer=${clayer##* }
	echo clayer: "${clayer}"

	# Delete the layer with no safety checks.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot delete $layer
	echo delete layer: "$output"
	[[ $status -eq 0 ]]

	# Delete the image while we're at it.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot delete $image
	echo delete image: "$output"
	[[ $status -eq 0 ]]

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Check and repair.  Repair is okay with deleting images because they
	# can be rebuilt or re-pulled.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r
	echo "check -r:" "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "container ${container}:" ]]
	[[ $output =~ "image ${image}: image missing" ]]
	[[ $output =~ "layer ${clayer} used by container ${container}: layer is in use by a container" ]]

	# couldn't get rid of the container, even so

	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check
	echo check: "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "container ${container}:" ]]
	[[ $output =~ "image ${image}: image missing" ]]

	# Repair, but now we're okay with getting rid of damaged containers.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r -f
	echo "check -r -f:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "container ${container}:" ]]
	[[ $output =~ "image ${image}: image missing" ]]

	# Should be all clear now.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect layer data being lost.
@test "check-layer-data-missing" {
	# Create the layer.
	run storage --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the image.
	run storage --debug=false create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Set a data item associated with the layer.
	createrandom ${TESTDIR}/datafile
	storage set-layer-data -f ${TESTDIR}/datafile $layer datafile
	run storage --debug=false list-layer-data $layer
	echo list-layer-data: "$output"
	[[ $status -eq 0 ]]
	[[ $output != "" ]]

	# Everything should look okay.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]

	# Delete that content and see if we notice.
	rm -fv ${TESTDIR}/root/${STORAGE_DRIVER}-layers/$layer/datafile
	run storage check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "layer ${layer}: data item \"datafile\": layer data item is missing" ]]

	# Should have repaired by deleting the image and layer, so we should be
	# in the clear.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect read-only layer data being lost.
@test "check-ro-layer-data-missing" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Create the read-only layer.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the image.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Set a data item associated with the layer.
	createrandom ${TESTDIR}/datafile
	storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot set-layer-data -f ${TESTDIR}/datafile $layer datafile
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot list-layer-data $layer
	echo list-layer-data: "$output"
	[[ $status -eq 0 ]]
	[[ $output != "" ]]

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Create a read-write layer.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rwlayer=$output

	# Create a read-write image.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-image $rwlayer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Everything should look okay.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check
	echo check: "$output"
	[[ $status -eq 0 ]]

	# Delete that content and see if we notice.
	rm -fv ${TESTDIR}/ro-root/${STORAGE_DRIVER}-layers/$layer/datafile
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r
	echo "check -r:" "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "layer ${layer}: data item \"datafile\": layer data item is missing" ]]

	# Can't repair it by deleting the image and layer, so we should be just
	# as broken as last time.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check
	echo check: "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "layer ${layer}: data item \"datafile\": layer data item is missing" ]]

	# A check of just the read-write storage shouldn't turn up anything.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect image data being lost.
@test "check-image-data-missing" {
	# Create the layer.
	run storage --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the image.
	run storage --debug=false create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]
	image=$output

	# Create the data associated with the image.
	createrandom ${TESTDIR}/datafile
	storage set-image-data -f ${TESTDIR}/datafile $image datafile
	run storage --debug=false list-image-data $image
	echo list-image-data: "$output"
	[[ $status -eq 0 ]]
	[[ $output != "" ]]

	# Everything should look good so far.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]

	# Now delete the data and check that we notice it's gone.
	rm -fv ${TESTDIR}/root/${STORAGE_DRIVER}-images/$image/datafile
	run storage check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "image ${image}: data item \"datafile\": image data item is missing" ]]

	# Having repaired it by deleting the offending image, we should be okay again.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect image data in read-only stores being lost.
@test "check-ro-image-data-missing" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Create the read-only layer.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the image.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]
	image=$output

	# Create the data associated with the image.
	createrandom ${TESTDIR}/datafile
	storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot set-image-data -f ${TESTDIR}/datafile $image datafile
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot list-image-data $image
	echo list-image-data: "$output"
	[[ $status -eq 0 ]]
	[[ $output != "" ]]

	# Everything should look good so far.
	run storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot check
	echo check: "$output"
	[[ $status -eq 0 ]]

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Create a read-write layer.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rwlayer=$output

	# Create a read-write image.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-image $rwlayer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Now delete the data and check that we notice it's gone.
	rm -fv ${TESTDIR}/ro-root/${STORAGE_DRIVER}-images/$image/datafile
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r
	echo "check -r:" "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "image ${image}: data item \"datafile\": image data item is missing" ]]

	# Having been unable to repair it by deleting the offending image, we
	# should still flag the error.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check
	echo check: "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "image ${image}: data item \"datafile\": image data item is missing" ]]

	# A check of just the read-write storage shouldn't turn up anything.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect image data being modified.
@test "check-image-data-modified" {
	# Create the layer.
	run storage --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the image.
	run storage --debug=false create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]
	image=$output

	# Create some data to associate with the image.
	createrandom ${TESTDIR}/datafile
	storage set-image-data -f ${TESTDIR}/datafile $image datafile
	run storage --debug=false list-image-data $image
	echo list-image-data: "$output"
	[[ $status -eq 0 ]]
	[[ $output != "" ]]

	# Everything should look okay so far.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]

	# Corrupt that data and see if we notice.
	echo "" >> ${TESTDIR}/root/${STORAGE_DRIVER}-images/$image/datafile
	run storage check -r
	echo "check -r:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "image ${image}: data item \"datafile\": image data item has incorrect size" ]]

	# We fixed that by removing the image, so everything should be okay now.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect image data being modified in read-only locations.
@test "check-ro-image-data-modified" {
	case "$STORAGE_DRIVER" in
	overlay*|vfs)
		;;
	*)
		skip "driver $STORAGE_DRIVER does not support additional image stores"
		;;
	esac

	# Create the read-only layer.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	layer=$output

	# Create the read-only image.
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot create-image $layer
	echo create-image: "$output"
	[[ $status -eq 0 ]]
	image=$output

	# Create some data to associate with the read-only image.
	createrandom ${TESTDIR}/datafile
	storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot set-image-data -f ${TESTDIR}/datafile $image datafile
	run storage --debug=false --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot list-image-data $image
	echo list-image-data: "$output"
	[[ $status -eq 0 ]]
	[[ $output != "" ]]

	# Everything should look okay so far.
	run storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot check
	echo check: "$output"
	[[ $status -eq 0 ]]

        storage --graph ${TESTDIR}/ro-root --run ${TESTDIR}/ro-runroot shutdown

	# Create a read-write layer.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rwlayer=$output

	# Create a read-write image.
	run storage --debug=false --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root create-image $rwlayer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Corrupt that data and see if we notice.
	echo "" >> ${TESTDIR}/ro-root/${STORAGE_DRIVER}-images/$image/datafile
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check -r
	echo "check -r:" "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "image ${image}: data item \"datafile\": image data item has incorrect size" ]]

	# We couldn't fix that by removing the image, so we should still notice the problem.
	run storage --storage-opt ${STORAGE_DRIVER}.imagestore=${TESTDIR}/ro-root check
	echo check: "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "image ${image}: data item \"datafile\": image data item has incorrect size" ]]

	# A check of just the read-write storage shouldn't turn up anything.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect container data being lost.
@test "check-container-data-missing" {
	# Create a container that isn't using an image as its base.
	run storage --debug=false create-container ""
	echo create-container: "$output"
	[[ $status -eq 0 ]]
	container=$output

	# Store some data alongside the container.
	createrandom ${TESTDIR}/datafile
	storage set-container-data -f ${TESTDIR}/datafile $container datafile
	run storage --debug=false list-container-data $container
	echo list-container-data: "$output"
	[[ $status -eq 0 ]]
	[[ $output != "" ]]

	# Everything should look okay so far.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]

	# Now remove the associated data and see if we notice.
	rm -fv ${TESTDIR}/root/${STORAGE_DRIVER}-containers/$container/datafile
	run storage check -r
	echo "check -r:" "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "container ${container}: data item \"datafile\": container data item is missing" ]]

	# didn't get rid of the container, though

	# Should still look broken.
	run storage check
	echo check: "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "container ${container}: data item \"datafile\": container data item is missing" ]]

	# Now let repair remove containers.
	run storage check -r -f
	echo "check -r -f:" "$output"
	[[ $status -eq 0 ]]

	# Should look okay now.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}

# Check that we can detect container data being modified.
@test "check-container-data-modified" {
	# Create a container that isn't using an image as its base.
	run storage --debug=false create-container ""
	echo create-container: "$output"
	[[ $status -eq 0 ]]
	container=$output

	# Store some data alongside the container.
	createrandom ${TESTDIR}/datafile
	storage set-container-data -f ${TESTDIR}/datafile $container datafile
	run storage --debug=false list-container-data $container
	echo list-container-data: "$output"
	[[ $status -eq 0 ]]
	[[ $output != "" ]]

	# Everything should look okay so far.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]

	# Create a read-write layer.
	run storage --debug=false create-layer
	echo create-layer: "$output"
	[[ $status -eq 0 ]]
	rwlayer=$output

	# Create a read-write image.
	run storage --debug=false create-image $rwlayer
	echo create-image: "$output"
	[[ $status -eq 0 ]]

	# Now remove the associated data and see if we notice.
	echo "" >> ${TESTDIR}/root/${STORAGE_DRIVER}-containers/$container/datafile
	run storage check -r
	echo "check -r:" "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "container ${container}: data item \"datafile\": container data item has incorrect size" ]]

	# didn't get rid of the container, though

	# Should still look broken.
	run storage check
	echo check: "$output"
	[[ $status -ne 0 ]]
	[[ $output =~ "container ${container}: data item \"datafile\": container data item has incorrect size" ]]

	# Now let repair remove containers.
	run storage check -r -f
	echo "check -r -f:" "$output"
	[[ $status -eq 0 ]]
	[[ $output =~ "container ${container}: data item \"datafile\": container data item has incorrect size" ]]

	# Should look okay now.
	run storage check
	echo check: "$output"
	[[ $status -eq 0 ]]
}
