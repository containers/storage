package chrootarchive

import (
	"io"

	"github.com/containers/storage/pkg/archive"
)

func invokeUnpack(decompressedArchive io.Reader,
	dest string,
	options *archive.TarOptions, root string,
) error {
	return fmt.Errorf("cannot unpack via chroot on this platform")
}

func invokePack(srcPath string, options *archive.TarOptions, root string) (io.ReadCloser, error) {
	_ = root // Restricting the operation to this root is not implemented on macOS
	return archive.TarWithOptions(srcPath, options)
}
