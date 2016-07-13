This is cowman, which provides a mostly direct interface to the graph drivers,
providing ways to create, remove, mount, and unmount layers, as well as
comparing them and applying diffs to them, and adds minimal tracking of their
relationships to the mix.

It provides a notion of Images (a layer, and caller-supplied metadata which
could be a manifest) and Containers (a layer derived from an Image, and
caller-supplied metadata which could be configuration).
