## oci-storage-list-deps 1 "August 2016"

## NAME
oci-storage list-deps - List known layers

## SYNOPSIS
**oci-storage** [*options* [...]] *layerOrImageOrContainerNameOrID*

## DESCRIPTION
Retrieves and prints a list of items which either directly or indirectly depend
on the specified layer, image, or container, and which should be removed before
an attempt is made to remove the specified layer, image, or container.

## EXAMPLE
**oci-storage list-deps my-layer**

## SEE ALSO
oci-storage-layers(1)
