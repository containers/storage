## containers-storage 1 "August 2024"

## NAME
containers-storage-composefs - Information about composefs and containers/storage


## DESCRIPTION

To enable composefs at a baseline requires the following configuration in `containers-storage.conf`:

```
[storage.options.overlay]
use_composefs = "true"
```

However at the current time, composefs requires zstd:chunked images, so first
you must be sure that zstd:chunked is enabled. For more, see [zstd:chunked](containers-storage-zstd-chunked.md).

Additionally, not many images are in zstd:chunked format. In order to bridge this gap,
`convert_images = "true"` can be specified which does a dynamic conversion; this adds
latency to image pulls.

Putting these things together, the following is required (in addition to the above config).

```
pull_options = {enable_partial_images = "true", use_hard_links = "false", ostree_repos="", convert_images = "true"}
```

## BUGS

- https://github.com/containers/storage/issues?q=is%3Aissue+is%3Aopen+label%3Aarea%2Fcomposefs

## FOOTNOTES
The Containers Storage project is committed to inclusivity, a core value of open source.
The `master` and `slave` mount propagation terminology is used in this repository.
This language is problematic and divisive, and should be changed.
However, these terms are currently used within the Linux kernel and must be used as-is at this time.
When the kernel maintainers rectify this usage, Containers Storage will follow suit immediately.
