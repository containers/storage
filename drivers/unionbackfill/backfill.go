package unionbackfill

import (
	"archive/tar"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/system"
)

// NewBackfiller supplies a backfiller whose Backfill method provides the
// ownership/permissions/attributes of a directory from a lower layer so that
// we don't have to create it in an upper layer using default values that will
// be mistaken for a reason that the directory was pulled up to that layer.
func NewBackfiller(idmap *idtools.IDMappings, lowerDiffDirs []string) *backfiller {
	if idmap != nil {
		uidMaps, gidMaps := idmap.UIDs(), idmap.GIDs()
		if len(uidMaps) > 0 || len(gidMaps) > 0 {
			idmap = idtools.NewIDMappingsFromMaps(append([]idtools.IDMap{}, uidMaps...), append([]idtools.IDMap{}, gidMaps...))
		}
	}
	return &backfiller{idmap: idmap, lowerDiffDirs: append([]string{}, lowerDiffDirs...)}
}

type backfiller struct {
	idmap         *idtools.IDMappings
	lowerDiffDirs []string
}

// Backfill supplies the ownership/permissions/attributes of a directory from a
// lower layer so that we don't have to create it in an upper layer using
// default values that will be mistaken for a reason that the directory was
// pulled up to that layer.
func (b *backfiller) Backfill(pathname string) (*tar.Header, error) {
	for _, lowerDiffDir := range b.lowerDiffDirs {
		candidate := filepath.Join(lowerDiffDir, pathname)
		// if the asked-for path is in this lower, return a tar header for it
		if st, err := os.Lstat(candidate); err == nil {
			var linkTarget string
			if st.Mode()&fs.ModeType == fs.ModeSymlink {
				target, err := os.Readlink(candidate)
				if err != nil {
					return nil, err
				}
				linkTarget = target
			}
			hdr, err := tar.FileInfoHeader(st, linkTarget)
			if err != nil {
				return nil, err
			}
			// this is where we'd delete "opaque" from the header, if FileInfoHeader read xattrs
			hdr.Name = strings.Trim(filepath.ToSlash(pathname), "/")
			if st.Mode()&fs.ModeType == fs.ModeDir {
				hdr.Name += "/"
			}
			if b.idmap != nil && !b.idmap.Empty() {
				if uid, gid, err := b.idmap.ToContainer(idtools.IDPair{UID: hdr.Uid, GID: hdr.Gid}); err == nil {
					hdr.Uid, hdr.Gid = uid, gid
				}
			}
			return hdr, nil
		}
		// if the directory or any of its parents is marked opaque, we're done looking at lowers
		p := strings.Trim(pathname, "/")
		subpathname := ""
		for {
			dir, subdir := filepath.Split(p)
			dir = strings.Trim(dir, "/")
			if dir == p {
				break
			}
			// kernel overlay style
			xval, err := system.Lgetxattr(filepath.Join(lowerDiffDir, dir), archive.GetOverlayXattrName("opaque"))
			if err == nil && len(xval) == 1 && xval[0] == 'y' {
				return nil, nil
			}
			// aufs or fuse-overlayfs using aufs-like whiteouts
			if _, err := os.Stat(filepath.Join(lowerDiffDir, dir, archive.WhiteoutOpaqueDir)); err == nil {
				return nil, nil
			}
			// kernel overlay "redirect" - starting with the next lower layer, we'll need to look elsewhere
			subpathname = strings.Trim(path.Join(subdir, subpathname), "/")
			xval, err = system.Lgetxattr(filepath.Join(lowerDiffDir, dir), archive.GetOverlayXattrName("redirect"))
			if err == nil && len(xval) > 0 {
				subdir := string(xval)
				if path.IsAbs(subdir) {
					// path is relative to the root of the mount point
					pathname = path.Join(subdir, subpathname)
				} else {
					// path is relative to the current directory
					parent, _ := filepath.Split(dir)
					parent = strings.Trim(parent, "/")
					pathname = path.Join(parent, subdir, subpathname)
				}
				break
			}
			p = dir
		}
	}
	return nil, nil
}
