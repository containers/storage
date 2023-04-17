## containers-storage-check 1 "September 2022"

## NAME
containers-storage check - Check for and remove damaged layers/images/containers

## SYNOPSIS
**containers-storage** **check** [-q] [-r [-f]]

## DESCRIPTION
Checks layers, images, and containers for identifiable damage.

## OPTIONS

**-f**

When repairing damage, also remove damaged containers.  No effect unless *-r*
is used.

**-r**

Attempt to repair damage by removing damaged images and layers.  If not
specified, damage is reported but not acted upon.

**-q**

Perform only checks which are not expected to be time-consuming.  This
currently skips verifying that a layer which was initialized using a diff can
reproduce that diff if asked to.

## EXAMPLE
**containers-storage check -r -f

## SEE ALSO
containers-storage(1)
