## containers-storage-dedup 1 "November 2024"

## NAME
containers-storage dedup - Deduplicate similar files in the images

## SYNOPSIS
**containers-storage** **dedup**

## DESCRIPTION
Find similar files in the images and deduplicate them.  It requires reflink support from the file system.

## OPTIONS
**--hash-method** *method*

Specify the function to use to calculate the hash for a file.  It can be one of: *size*, *crc*, *sha256sum*.

## EXAMPLE
**containers-storage dedup**
