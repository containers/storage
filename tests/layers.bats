#!/usr/bin/env bats
# vim:set syn=bash:

load helpers

@test "allow storing images with more than 127 layers" {
    LAYER=""
    LIMIT=300
    for i in $(seq 0 ${LIMIT}); do
        echo "Layer: $i"

        # Create a layer.
        run storage --debug=false create-layer "$LAYER"
        echo "$output"
        [ "$status" -eq 0 ]
        [ "$output" != "" ]
        LAYER="$output"
        run storage --debug=false mount "$LAYER"
        echo "$output"
        [ "$status" -eq 0 ]
        [ "$output" != "" ]
        ROOTFS="$output"
        touch "${ROOTFS}"/$i
        run storage --debug=false unmount "$LAYER"
        echo "$output"
        [ "$status" -eq 0 ]
    done

    # Create the image
    run storage --debug=false create-image "$LAYER"
    echo "$output"
    [ "$status" -eq 0 ]
    [ "$output" != "" ]
    IMAGE="$output"

    # Make sure the image has all of the content.
    run storage --debug=false create-container "$IMAGE"
    echo "$output"
    [ "$status" -eq 0 ]
    [ "$output" != "" ]
    CONTAINER="$output"

    run storage --debug=false mount "$CONTAINER"
    echo "$output"
    [ "$status" -eq 0 ]
    [ "$output" != "" ]
    ROOTFS="$output"
    for i in $(seq 0 ${LIMIT}); do
        if ! test -r "${ROOTFS}"/$i ; then
            echo File from layer $i of ${LIMIT} was not visible after mounting
            false
        fi
    done

    run storage --debug=false unmount "$CONTAINER"
    echo "$output"
    [ "$status" -eq 0 ]
}
