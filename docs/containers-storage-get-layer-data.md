## containers-storage-get-layer-data 1 "December 2020"

## NAME
containers-storage get-layer-data - Retrieve lookaside data for a layer

## SYNOPSIS
**containers-storage** **get-layer-data** [*options* [...]] *layerID* *dataName*

## DESCRIPTION
Retrieves a piece of named data which is associated with a layer.

## OPTIONS
**-f | --file** *file*

Write the data to a file instead of stdout.

## EXAMPLE
**containers-storage get-layer-data -f config.json 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824 configuration**

## SEE ALSO
containers-storage-set-layer-data(1)
