This is graphtool, a tool which can handle creating, removing, mounting, and
unmounting of read-write layers for modification by the host, and which can
handle importing and exporting docker images.

For lower level cases, we also provide a more direct interface to the graph
drivers, providing ways to create, remove, mount, and unmount layers, as well
as comparing them and applying diffs to them.
