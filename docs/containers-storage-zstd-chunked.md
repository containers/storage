## containers-storage 1 "August 2024"

## NAME
containers-storage-zstd-chunked - Information about zstd:chunked


## DESCRIPTION

The traditional format for container image layers is [application/vnd.oci.image.layer.v1.tar+gzip](https://github.com/opencontainers/image-spec/blob/main/layer.md#gzip-media-types).
More recently, the standard was augmented with zstd: [application/vnd.oci.image.layer.v1.tar+zstd](https://github.com/opencontainers/image-spec/blob/main/layer.md#zstd-media-types)
which is a more modern and efficient compression format.

`zstd:chunked` is a variant of the `application/vnd.oci.image.layer.v1.tar+zstd` media type that
uses zstd [skippable frames](https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md#skippable-frames)
to include additional metadata (especially a "table of contents") that includes the SHA-256 and offsets of individual chunks of files.
Additionally chunks are compressed separately. This allows a client to dynamically fetch only content which
it doesn't already have using HTTP range requests.

At the time of this writing, support for this is enabled by default in the code.

You can explicitly enable or disable zstd:chunked with following changes to `containers-storage.conf`:

```
[storage.overlay.pull_options]
enable_partial_images = "true" | "false"
```

Note that the value of this field must be a "string bool", it cannot be a native TOML boolean.

## INTERNALS

At the current time the format is not officially standardized or documented beyond
the comments and code in the reference implementation. At the current time
the file with the most information is [pkg/chunked/internal/compression.go](https://github.com/containers/storage/blob/39d469c34c96db67062e25954bc9d18f2bf6dae3/pkg/chunked/internal/compression.go).
The above is a permanent link for stability, but be sure to check to see if there are newer changes too.

## BUGS

- https://github.com/containers/storage/issues?q=is%3Aissue+label%3Aarea%2Fzstd%3Achunked+is%3Aopen

## FOOTNOTES
The Containers Storage project is committed to inclusivity, a core value of open source.
The `master` and `slave` mount propagation terminology is used in this repository.
This language is problematic and divisive, and should be changed.
However, these terms are currently used within the Linux kernel and must be used as-is at this time.
When the kernel maintainers rectify this usage, Containers Storage will follow suit immediately.
