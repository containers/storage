#!/usr/bin/env bats
# vim:set syn=bash:

load helpers

@test "allow storing images with more than 127 layers" {
    LAYER=""
    for i in {0..150}; do
        echo "Layer: $i"

        # Create a layer.
        run storage --debug=false create-layer "$LAYER"
        [ "$status" -eq 0 ]
        [ "$output" != "" ]
        LAYER="$output"
    done

    # Create the image
    run storage --debug=false create-image --name test-image "$LAYER"
    [ "$status" -eq 0 ]
    [ "$output" != "" ]
}
