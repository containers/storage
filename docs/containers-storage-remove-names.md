## containers-storage-remove-names 1 "January 2023"

## NAME
containers-storage remove-names - Remove names from a layer/image/container

## SYNOPSIS
**containers-storage** **remove-names** [*options* [...]] *layerOrImageOrContainerNameOrID*

## DESCRIPTION
In addition to IDs, *layers*, *images*, and *containers* can have
human-readable names assigned to them in *containers-storage*.  The *remove-names*
command can be used to remove one or more names from them.

## OPTIONS
**-n | --name** *name*

Specifies a name to remove from the layer, image, or container.

## EXAMPLE
**containers-storage remove-names -n my-for-realsies-awesome-container f3be6c6134d0d980936b4c894f1613b69a62b79588fdeda744d0be3693bde8ec**

## SEE ALSO
containers-storage-add-names(1)
containers-storage-get-names(1)
containers-storage-set-names(1)
