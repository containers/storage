## containers-storage-create-storage-layer 1 "September 2022"

## NAME
containers-storage create-storage-layer - Create a layer in a lower-level storage driver

## SYNOPSIS
**containers-storage** **create-storage-layer** [*options* [...]] [*parentLayerNameOrID*]

## DESCRIPTION
Creates a new layer using the lower-level storage driver which either has a
specified layer as its parent, or if no parent is specified, is empty.

## OPTIONS
**-i | --id** *ID*

Sets the ID for the layer.  If none is specified, one is generated.

**-l | --label** *mount-label*

Sets the label which should be assigned as an SELinux context when mounting the
layer.

## EXAMPLE
**containers-storage create-storage-layer somelayer**

## SEE ALSO
containers-storage-create-container(1)
containers-storage-create-image(1)
containers-storage-delete-layer(1)
