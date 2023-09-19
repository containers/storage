## containers-storage-applydiff-using-staging-dir 1 "September 2023"

## NAME
containers-storage applydiff-using-staging-dir - Apply a layer diff to a layer using a staging directory

## SYNOPSIS
**containers-storage** **applydiff-using-staging-dir** *layerNameOrID* *source*

## DESCRIPTION
When a layer is first created, it contains no changes relative to its parent
layer.  The layer can either be mounted read-write and its contents modified
directly, or contents can be added (or removed) by applying a layer diff.  A
layer diff takes the form of a (possibly compressed) tar archive with
additional information present in its headers, and can be produced by running
*containers-storage diff* or an equivalent.

Differently than **apply-diff**, the command **applydiff-using-staging-dir**
first creates a staging directory and then moves the final result to the destination.

## EXAMPLE
**containers-storage applydiff-using-staging-dir 5891b5b522 /path/to/diff**

## SEE ALSO
containers-storage-apply-diff(1)
containers-storage-changes(1)
containers-storage-diff(1)
containers-storage-diffsize(1)
