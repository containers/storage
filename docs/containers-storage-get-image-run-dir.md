## containers-storage-get-image-run-dir 1 "January 2024"

## NAME
containers-storage get-image-run-dir - Find runtime lookaside directory for an image

## SYNOPSIS
**containers-storage** **get-image-run-dir** [*options* [...]] *imageNameOrID*

## DESCRIPTION
Prints the location of a directory which the caller can use to store lookaside
information which should be cleaned up when the host is rebooted.

## EXAMPLE
**containers-storage get-image-run-dir my-image**

## SEE ALSO
containers-storage-get-image-dir(1)
