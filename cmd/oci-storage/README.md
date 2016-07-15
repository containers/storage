This is cowman.  Don't worry, that's a temporary name.

It depends on 'cow', which is a pretty barebones wrapping of the graph
drivers that exposes the create/mount/unmount/delete operations and adds
enough bookkeeping to know about the relationships between layers.

On top of that, it provides a way to mark a layer as an image, which
allows an API caller to attach an arbitrary blob of data to it, or as a
container, where in addition to noting which image was used to create
the container, it allows an API caller to attach an arbitrary blob to
it.

Layers, images, and containers are all identified using an ID which can
be set when they are created, and can optionally be assigned names which
are resolved to IDs automatically by the APIs.

The cowman tool is a CLI that wraps that as thinly as possible, so that
other tooling can use it to import layers from images.  Those other
tools can then either manage the concept of images on their own, or let
the API/CLI handle storing the image metadata and/or configuration.
Likewise, other tools can create container layers and manage them on
their own or use the API/CLI for storing what I assume will be container
metadata and/or configurations.

Logic for importing images and creating and managing containers will
most likely be implemented elsewhere, and if that implementation ends up
not needing the API/CLI to provide a place to store data about images
and containers, that functionality can be dropped.
