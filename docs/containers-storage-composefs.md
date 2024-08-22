## containers-storage 1 "August 2024"

## NAME
containers-storage-composefs - Information about composefs and containers/storage

## DESCRIPTION

To enable composefs at a baseline requires the following configuration in `containers-storage.conf`:

```
[storage.options.overlay]
use_composefs = "true"
```

This value must be a "string bool", it cannot be a native TOML boolean.

However at the current time, composefs requires zstd:chunked images, so first
you must be sure that zstd:chunked is enabled. For more, see [zstd:chunked](containers-storage-zstd-chunked.md).

Additionally, not many images are in zstd:chunked format. In order to bridge this gap,
`convert_images = "true"` can be specified which does a dynamic conversion; this adds
latency to image pulls.

Putting these things together, the following is required (in addition to the above config).

```
[storage.options.pull_options]
convert_images = "true"
```

This value must be a "string bool", it cannot be a native TOML boolean.

## IMPLEMENTATION

As is implied by the setting `use_composefs = "true"`, currently composefs
is implemented as an "option" for the `overlay` driver. Some file formats
remain unchanged and are inherited from the overlay driver, even when
composefs is in use. The primary differences are enumerated below.

The `diff/` directory for each layer is no longer a plain unpacking of the tarball,
but becomes an "object hash directory", where each filename is the sha256 of its contents. This `diff/`
directory is the backing store for a `composefs-data/composefs.blob` created for
each layer which is the composefs "superblock" containing all the non-regular-file content (i.e. metadata) from the tarball.

As with `zstd:chunked`, existing layers are scanned for matching objects, and reused
(via hardlink or reflink as configured) if objects with a matching "full sha256" are
found.

There is currently no support for enforced integrity with composefs;
an attempt is made to enable fsverity for the backing files and the composefs file,
but it is not an error if unsupported. There is as of yet no defined mechanism to
verify the fsverity digest of the composefs block before mounting; some work on that is
ongoing.

In order to mount a layer (or a full image, with all of its dependencies), any
layer that has a composefs blob is mounted and included in the "final" overlayfs
stack. This is optional - any layers that are not in "composefs format" but
in the "default overlay" (unpacked) format will be reused as is.

## BUGS

- https://github.com/containers/storage/issues?q=is%3Aissue+is%3Aopen+label%3Aarea%2Fcomposefs

## FOOTNOTES
The Containers Storage project is committed to inclusivity, a core value of open source.
The `master` and `slave` mount propagation terminology is used in this repository.
This language is problematic and divisive, and should be changed.
However, these terms are currently used within the Linux kernel and must be used as-is at this time.
When the kernel maintainers rectify this usage, Containers Storage will follow suit immediately.
