## containers-storage-gc 1 "January 2023"

## NAME
containers-storage gc - Garbage collect leftovers from partial layers/images/contianers

## SYNOPSIS
**containers-storage** **gc**

## DESCRIPTION
Removes additional data for layers, images, and containers which would
correspond to layers, images, and containers which don't actually exist, but
which may have been left on the filesystem after canceled attempts to create
those layers, images, or containers.

## EXAMPLE
**containers-storage gc**
