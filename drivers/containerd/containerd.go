// +build linux

package containerd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/leases"
	mount "github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/snapshots"
	graphdriver "github.com/containers/storage/drivers"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/containers/storage/pkg/locker"
	"github.com/containers/storage/pkg/parsers"
	"github.com/containers/storage/pkg/stringid"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	driverName = "containerd"

	// namespace used by this graphdriver in containerd
	namespace = "containers"

	// additional layer information passed by image package
	labelTargetDiffID      = "containers/image/target.diffID"
	labelTargetDigest      = "containers/image/target.layerdigest"
	labelTargetReference   = "containers/image/target.reference"
	labelTargetImageLayers = "containers/image/target.layers"

	// labels used for leveraging "remote" snapshots.
	labelSnapshotRef              = "containerd.io/snapshot.ref"
	labelSnapshotDigest           = "containerd.io/snapshot/containers.digest"
	labelSnapshotImageReference   = "containerd.io/snapshot/containers.reference"
	labelSnapshotImageLayers      = "containerd.io/snapshot/containers.layers"
	labelSnapshotContentNamespace = "containerd.io/snapshot/containers.content-namespace"
	labelSnapshotContentDiffID    = "containerd.io/snapshot/containers.content-diffID"
	labelSnapshotID               = "containerd.io/snapshot/containers/id.snapshot"
	labelContentDigest            = "containerd.io/snapshot/containers/digest.content"

	// labels used for resource management.
	labelGCRoot       = "containerd.io/gc.root"
	labelGCRefContent = "containerd.io/gc.ref.content"
)

// Driver contains information about the home directory and the list of mounts and
// contents that are created using this driver.
type Driver struct {
	home    string
	uidMaps []idtools.IDMap
	gidMaps []idtools.IDMap

	store           content.Store
	snapshotter     snapshots.Snapshotter
	snapshotterName string
	withLease       withLeaseFunc

	locker    *locker.Locker
	naiveDiff graphdriver.DiffDriver

	additionalStore string
	layerfile       string
}

var logger = logrus.WithField("storage-driver", driverName)

func init() {
	graphdriver.Register(driverName, Init)
}

// Init returns the contaierd-based graphdriver. This driver also
// leverages content store for contents management corresponding to
// snapshots.
func Init(home string, options graphdriver.Options) (graphdriver.Driver, error) {
	o, err := parseOptions(options.DriverOptions)
	if err != nil {
		return nil, err
	} else if o == nil || o.address == "" {
		return nil, fmt.Errorf("containerd address must be provided")
	}

	// TODO:
	// - more gRPC options if needed
	// - we might somtimes need reconnections during the execution of this driver.
	ctd, err := containerd.New(o.address)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to containerd on %q", o.address)
	}
	sn := ctd.SnapshotService(o.snapshotter)
	cs := ctd.ContentStore()
	wl := withLeaseFuncFromContainerd(ctd)
	uidMaps := options.UIDMaps
	gidMaps := options.GIDMaps
	rootUID, rootGID, err := idtools.GetRootUIDGID(uidMaps, gidMaps)
	if err != nil {
		return nil, err
	}
	// Create the driver home dir
	if err := idtools.MkdirAllAndChown(home, 0700, idtools.IDPair{UID: rootUID, GID: rootGID}); err != nil {
		return nil, err
	}

	// Prepare additional store directories
	additionalStore := filepath.Join(home, "internal-store")
	if err := os.MkdirAll(filepath.Join(additionalStore, driverName+"-layers"), 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(additionalStore, driverName+"-images"), 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(additionalStore, driverName+"-containers"), 0700); err != nil {
		return nil, err
	}

	d := &Driver{
		home:            home,
		uidMaps:         uidMaps,
		gidMaps:         gidMaps,
		store:           cs,
		snapshotter:     sn,
		snapshotterName: o.snapshotter,
		locker:          locker.New(),
		withLease:       wl,
		additionalStore: additionalStore,
		layerfile:       filepath.Join(additionalStore, driverName+"-layers", "layers.json"),
	}

	d.naiveDiff = graphdriver.NewNaiveDiffDriver(d, graphdriver.NewNaiveLayerIDMapUpdater(d))

	return d, nil
}

type containerdOptions struct {
	address     string
	snapshotter string
	// TODO: gRPC-related options if needed
}

func parseOptions(options []string) (*containerdOptions, error) {
	o := &containerdOptions{}
	for _, option := range options {
		key, val, err := parsers.ParseKeyValueOpt(option)
		if err != nil {
			return nil, err
		}
		key = strings.ToLower(key)
		switch key {
		case "address":
			o.address = val
		case "snapshotter":
			o.snapshotter = val
		default:
			return nil, fmt.Errorf("containerd: Unknown option %s", key)
		}
	}
	return o, nil
}

// String retuns the name of this graphdriver.
func (d *Driver) String() string {
	return driverName
}

// Status returns information about the background snapshotter.
func (d *Driver) Status() [][2]string {
	return [][2]string{
		{"Backing Snapshotter", d.snapshotterName},
	}
}

// GetMetadata returns information about snapshot and content binded to
// the specified layer ID.
func (d *Driver) Metadata(id string) (map[string]string, error) {
	d.locker.Lock(id)
	defer d.locker.Unlock(id)

	var (
		info = make(map[string]string)
		ctx  = namedCtx() // TODO: timeout?
	)

	// Get the information about snapshot
	sID, err := d.getSnapshotIDLock(id)
	if err != nil {
		return nil, err
	}
	sinfo, err := d.snapshotter.Stat(ctx, sID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to stat snapshot %q(key=%q)", id, sID)
	}
	info["SnapshotKind"] = sinfo.Kind.String()
	info["SnapshotName"] = sinfo.Name
	info["SnapshotParent"] = sinfo.Parent
	info["SnapshotCreated"] = sinfo.Created.String()
	info["SnapshotUpdated"] = sinfo.Updated.String()
	info["SnapshotLabels"] = ""
	for k, v := range sinfo.Labels {
		info["SnapshotLabels"] += fmt.Sprintf("%q: %q, ", k, v)
	}

	// Get the information about content
	if st, err := d.store.Status(ctx, id); err == nil {
		// Get the written-in-progress information
		info["ContentRef"] = st.Ref
		info["ContentOffset"] = fmt.Sprintf("%d", st.Offset)
		info["ContentTotal"] = fmt.Sprintf("%d", st.Total)
		info["ContentExpected"] = st.Expected.String()
		info["ContentStartedAt"] = st.StartedAt.String()
		info["ContentUpdatedAt"] = st.UpdatedAt.String()
	} else {
		cID, err := d.getContentIDLock(id)
		if err != nil {
			return nil, err
		}
		dgst, err := digest.Parse(cID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse registered cID %q for layer %q", cID, id)
		}
		// Get the committed content information
		cinfo, err := d.store.Info(ctx, dgst)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get info of %q for layer %q", dgst, id)
		}
		info["ContentDigest"] = cinfo.Digest.String()
		info["ContentSize"] = fmt.Sprintf("%d", cinfo.Size)
		info["ContentCreatedAt"] = cinfo.CreatedAt.String()
		info["ContentUpdatedAt"] = cinfo.UpdatedAt.String()
		info["ContentLabels"] = ""
		for k, v := range cinfo.Labels {
			info["ContentLabels"] += fmt.Sprintf("%q: %q, ", k, v)
		}
	}

	return info, nil
}

// Cleanup any state created by the snapshotter when daemon is being shutdown.
func (d *Driver) Cleanup() error {
	if c, ok := d.snapshotter.(snapshots.Cleaner); ok {
		return c.Cleanup(namedCtx())
	}
	return nil
}

// CreateReadWrite creates a new, empty snapshot that is ready to be used as
// the storage for a container.
func (d *Driver) CreateReadWrite(id, parent string, opts *graphdriver.CreateOpts) error {
	return d.create(id, parent, opts)
}

// Create creates a new, empty snapshot with the specified id and parent and
// options passed in opts.
func (d *Driver) Create(id, parent string, opts *graphdriver.CreateOpts) error {
	return d.create(id, parent, opts)
}

// CreateFromTemplate creates a layer with the same contents and parent as another layer.
func (d *Driver) CreateFromTemplate(id, template string, templateIDMappings *idtools.IDMappings, parent string, parentIDMappings *idtools.IDMappings, opts *graphdriver.CreateOpts, readWrite bool) error {
	if readWrite {
		return d.CreateReadWrite(id, template, opts)
	}
	return d.Create(id, template, opts)
}

func (d *Driver) create(id, parent string, opts *graphdriver.CreateOpts) error {
	d.locker.Lock(id)
	defer d.locker.Unlock(id)

	if _, err := d.getSnapshotIDLock(id); err == nil {
		return fmt.Errorf("layer (id=%q) already exists", id)
	}

	ctx, done, err := d.withLease(namedCtx()) // TODO: timeout?
	if err != nil {
		return err
	}
	defer done(ctx)

	var psID string
	if parent != "" {
		psID, err = d.getSnapshotIDLock(parent)
		if err != nil {
			return errors.Wrapf(err, "parent snapshot isn't registered for given id %q", parent)
		}
		info, err := d.snapshotter.Stat(ctx, psID)
		if err != nil {
			return errors.Wrapf(err, "failed to stat parent layer %q(key=%q)",
				parent, psID)
		}

		// If the parent snapshot hasn't been committed yet, commit it now so that
		// we can add a new snapshot on the top of it.
		// TODO1: This snapshot still should be able to be modified? Committed snapshot
		//        can't be modified anymore.
		// TODO2: The original active snapshot isn't accessible after this commit. This
		//        means if the user references the mountpoint of the active snapshot there
		//        is no guarantee that the mountpoint will be still accssesible.
		//        See also: https://github.com/containerd/containerd/commit/5e8218a63b468ea7ca19fe043c109cda45784570
		if info.Kind != snapshots.KindCommitted {
			labels := info.Labels
			if labels == nil {
				labels = make(map[string]string)
			}

			// We manually manage the lifecycle of this resource.
			labels[labelGCRoot] = time.Now().UTC().Format(time.RFC3339)
			labels[labelSnapshotID] = parent

			// Refer the content corresponding to this snapshot for preventing the content
			// getting deleted by GC.
			if pcID, err := d.getContentIDLock(id); err == nil {
				labels[labelGCRefContent] = pcID
			}

			var newname string
			for i := 0; i < 3; i++ {
				if newname, err = uniqueKey(); err != nil {
					continue
				}
				if err = d.snapshotter.Commit(ctx, newname, psID, snapshots.WithLabels(labels)); err == nil {
					break
				} else if err != nil && !errdefs.IsAlreadyExists(err) {
					return errors.Wrapf(err, "failed to commit parent layer %q(key=%q)", parent, psID)
				}
				// Key conflicts. try with other key
			}
			psID = newname
		}
	}

	var (
		labels       map[string]string
		targetDigest string
		targetDiffID string
	)
	if opts != nil && opts.StorageOpt != nil {
		targetDigest = opts.StorageOpt[labelTargetDigest]
		targetDiffID = opts.StorageOpt[labelTargetDiffID]
		labels = map[string]string{
			// The basic information of targetting snapshot
			labelSnapshotRef:            id,
			labelSnapshotDigest:         targetDigest,
			labelSnapshotImageReference: opts.StorageOpt[labelTargetReference],
			labelSnapshotImageLayers:    opts.StorageOpt[labelTargetImageLayers],

			// The following labels helps snapshotter to prepare contents from backing remote
			// storages and enables us to refer these contents later.
			labelSnapshotContentNamespace: namespace,
			labelSnapshotContentDiffID:    targetDiffID,

			// Refer to the content corresponding to this snapshot for preventing the
			// content getting deleted by GC.
			labelGCRefContent: targetDiffID,
		}
	} else {
		labels = make(map[string]string)
	}
	// We manually manage the lifecycle of this resource.
	labels[labelGCRoot] = time.Now().UTC().Format(time.RFC3339)
	labels[labelSnapshotID] = id

	// Preapre snapshot
	var sID string
	for i := 0; i < 3; i++ {
		var err error
		if sID, err = uniqueKey(); err != nil {
			continue
		}
		if _, err = d.snapshotter.Prepare(ctx, sID, psID, snapshots.WithLabels(labels)); err == nil {
			// Succeeded to prepare
			return nil
		} else if err != nil && !errdefs.IsAlreadyExists(err) {
			// Failed to prepare
			return errors.Wrapf(err, "failed to prepare snapshot %q for layer %q", sID, id)
		}

		// We are getting already exists error.
		// The possible reasons could be the following so we need to figure out which one here.
		// - Key conflicts.
		// - The snapshot is provided by snapshotter by the ChainID.
		var info snapshots.Info
		if info, err = d.snapshotter.Stat(ctx, id); err != nil {
			continue // Key conflicts. try with other key
		}

		// This layer is provided by snapshotter. Check the content existence.
		sID, err = id, nil
		// TODO: Check the Expected digest here but currently containerd doesn't support
		//       getting `Expected` field by `Status`.
		if _, err = d.store.Status(ctx, id); err != nil /* || st.Expected.String() != diffID.String() */ {
			// Corresponding content isn't written-in-progress.
			// Let's check if the content exists as a commtted content.
			if _, err = d.store.Info(ctx, digest.Digest(targetDiffID)); err != nil {
				d.snapshotter.Remove(ctx, sID)
				return errors.Wrapf(err, "failed to get content (digest=%q) for layer %q", targetDiffID, id)
			}
		}

		// The content exists in the backing content store.
		if info.Labels == nil {
			info.Labels = make(map[string]string)
		}
		info.Labels[labelContentDigest] = targetDiffID
		if _, err := d.snapshotter.Update(ctx, info, "labels."+labelContentDigest); err != nil {
			d.snapshotter.Remove(ctx, sID)
			return errors.Wrap(err, "failed to register content digest during preparing snapshot")
		}

		// Tell the client that this layer exists
		if err := d.addLayerToStoreLock(&Layer{
			ID:                 id,
			Parent:             parent,
			Created:            time.Now(),
			CompressedDigest:   digest.Digest(targetDigest),
			CompressedSize:     0,
			UncompressedDigest: digest.Digest(targetDiffID),
			UncompressedSize:   0,
			CompressionType:    archive.Gzip,
			ReadOnly:           true,
		}); err != nil {
			return err
		}
		return graphdriver.ErrTargetLayerAlreadyExists
	}

	return errors.Wrapf(err, "failed to create layer %q", id)
}

// Remove attempts to remove the snapshot and the contents corresponding to the layer.
func (d *Driver) Remove(id string) error {
	d.locker.Lock(id)
	defer d.locker.Unlock(id)

	sID, err := d.getSnapshotIDLock(id)
	if err != nil {
		return errors.Wrapf(err, "remove: snapshot doesn't registered for given id %q", id)
	}
	// The referencing content will be garbage collected by containerd
	if err := d.snapshotter.Remove(namedCtx(), sID); err != nil {
		return errors.Wrapf(err, "failed to remove snapshot %q(key=%q)", id, sID)
	}
	return nil
}

// Get returns the mountpoint for the snapshot referred to by this id.
func (d *Driver) Get(id string, options graphdriver.MountOpts) (dir string, retErr error) {
	d.locker.Lock(id)
	defer d.locker.Unlock(id)

	sID, err := d.getSnapshotIDLock(id)
	if err != nil {
		return "", errors.Wrapf(err, "get: snapshot doesn't registered for given id %q", id)
	}

	ctx, done, err := d.withLease(namedCtx()) // TODO: timeout?
	if err != nil {
		return "", err
	}
	defer done(ctx)

	dir = d.dir(id)
	rootUID, rootGID, err := idtools.GetRootUIDGID(d.uidMaps, d.gidMaps)
	if err != nil {
		return "", err
	}
	if err := idtools.MkdirAndChown(dir, 0700, idtools.IDPair{UID: rootUID, GID: rootGID}); err != nil {
		return "", err
	}

	defer func() {
		if retErr != nil {
			if rmErr := unix.Rmdir(dir); rmErr != nil && !os.IsNotExist(rmErr) {
				logger.Debugf("Failed to remove %s: %v: %v", id, rmErr, err)
			}
		}
	}()

	info, err := d.snapshotter.Stat(ctx, sID)
	if err != nil {
		retErr = err
		return
	}
	var m []mount.Mount
	if info.Kind == snapshots.KindActive {
		if m, retErr = d.snapshotter.Mounts(ctx, sID); retErr != nil {
			return
		}
	} else {
		if info.Labels == nil {
			info.Labels = make(map[string]string)
		}
		labelGCRefSnapshot := fmt.Sprintf("containerd.io/gc.ref.snapshot.%s", d.snapshotterName)

		// readonly view
		for i := 0; i < 3; i++ {
			var vKey string
			vKey, retErr = uniqueKey()
			if retErr != nil {
				continue
			}
			// reference the view for the original snapshot so that the view won't
			// be removed by GC.
			info.Labels[labelGCRefSnapshot] = vKey
			if _, retErr = d.snapshotter.Update(ctx, info, "labels."+labelGCRefSnapshot); retErr != nil {
				retErr = errors.Wrap(err, "failed to configure GC")
				return
			}
			if _, retErr = d.snapshotter.View(ctx, vKey, sID); retErr == nil || !errdefs.IsAlreadyExists(retErr) {
				break
			}
			// Key conflicts. try with other key
		}
		if retErr != nil {
			return
		}
	}
	if err := mount.All(m, dir); err != nil {
		retErr = err
		return
	}
	return dir, nil
}

// Put unmounts the mount path created for the give id.
func (d *Driver) Put(id string) error {
	d.locker.Lock(id)
	defer d.locker.Unlock(id)

	dir := d.dir(id)
	if err := mount.Unmount(dir, 0); err != nil {
		return errors.Wrapf(err, "failed to unmount layer %q on %q", id, dir)

	}
	if err := unix.Rmdir(dir); err != nil && !os.IsNotExist(err) {
		logger.Debugf("Failed to remove %s: %v", id, err)
	}
	return nil
}

// Exists returns whether a layer with the specified ID exists on this driver.
func (d *Driver) Exists(id string) bool {
	d.locker.Lock(id)
	defer d.locker.Unlock(id)

	sID, err := d.getSnapshotIDLock(id)
	if err != nil {
		return false
	}
	_, err = d.snapshotter.Stat(namedCtx(), sID)
	return err == nil
}

// AdditionalImageStores returns additional image stores supported by the driver
func (d *Driver) AdditionalImageStores() []string {
	return []string{d.additionalStore}
}

// ApplyDiff applies the new layer into a root
func (d *Driver) ApplyDiff(id string, parent string, options graphdriver.ApplyDiffOpts) (size int64, err error) {
	if !d.isParent(id, parent) {
		return d.naiveDiff.ApplyDiff(id, parent, options)
	}

	diff := options.Diff
	d.locker.Lock(id)
	if _, err := d.getContentIDLock(id); err == nil {
		d.locker.Unlock(id)
		return 0, fmt.Errorf("applying diff to %q on %q twice isn't supported", id, parent)
	}
	d.locker.Unlock(id)

	ctx, done, err := d.withLease(namedCtx()) // TODO: timeout?
	if err != nil {
		return 0, err
	}
	defer done(ctx)

	// Open the content writer and provide the diff stream to the
	// content store as well as applying the diff to the targetting snapshot
	cw, err := content.OpenWriter(ctx, d.store, content.WithRef(id))
	if err != nil {
		return 0, err
	}
	defer cw.Close()
	digester := digest.Canonical.Digester()
	dr := io.TeeReader(diff, io.MultiWriter(cw, digester.Hash()))
	options.Diff = dr
	applySize, err := d.naiveDiff.ApplyDiff(id, parent, options)
	if err != nil {
		return 0, err
	}
	io.Copy(ioutil.Discard, dr) // makes sure all contents to be read

	// Configure the prepared snapshot
	d.locker.Lock(id)
	defer d.locker.Unlock(id)
	sID, err := d.getSnapshotIDLock(id)
	if err != nil {
		if aErr := d.store.Abort(ctx, id); aErr != nil {
			logger.Debugf("Failed to abort %s: %v", id, aErr)
		}
		return 0, errors.Wrapf(err, "snapshot %q isn't registered for %q", id, parent)
	}
	diffID := digester.Digest()
	info, err := d.snapshotter.Stat(ctx, sID)
	if err != nil {
		return 0, err
	}
	if info.Labels == nil {
		info.Labels = make(map[string]string)
	}
	info.Labels[labelGCRefContent] = diffID.String()
	if _, err := d.snapshotter.Update(ctx, info, "labels."+labelGCRefContent); err != nil {
		if aErr := d.store.Abort(ctx, id); aErr != nil {
			logger.Debugf("Failed to abort %s: %v", id, aErr)
		}
		return 0, errors.Wrap(err, "failed to configure GC")
	}

	// Finally, commit the provided diff contents
	if err := cw.Commit(ctx, 0, diffID); err != nil && !errdefs.IsAlreadyExists(err) {
		if aErr := d.store.Abort(ctx, id); aErr != nil {
			logger.Debugf("Failed to abort %s: %v", id, aErr)
		}
		return 0, err
	}
	info.Labels[labelContentDigest] = diffID.String()
	if _, err := d.snapshotter.Update(ctx, info, "labels."+labelContentDigest); err != nil {
		return 0, errors.Wrap(err, "failed to register content digest during applying diff")
	}

	return applySize, nil
}

// DiffSize calculates the changes between the specified id
// and its parent and returns the size in bytes of the changes
// relative to its base filesystem directory.
func (d *Driver) DiffSize(id string, idMappings *idtools.IDMappings, parent string, parentIDMappings *idtools.IDMappings, mountLabel string) (size int64, err error) {
	if !d.isParent(id, parent) {
		return d.naiveDiff.DiffSize(id, idMappings, parent, parentIDMappings, mountLabel)
	}

	d.locker.Lock(id)
	defer d.locker.Unlock(id)

	// get the size in the snapshotter
	key, err := d.getSnapshotIDLock(id)
	if err != nil {
		return 0, errors.Wrapf(err, "diffsize: snapshot doesn't registered for given id %q", id)
	}
	ctx := namedCtx() // TODO: timeout?
	usage, err := d.snapshotter.Usage(ctx, key)
	if err != nil {
		return 0, err
	}
	size += usage.Size

	// get the size in the content store (if exists the contents)
	if status, err := d.store.Status(ctx, id); err == nil {
		size += status.Total // This is in progress so can be changed
	}
	if cID, err := d.getContentIDLock(id); err == nil {
		dgst, err := digest.Parse(cID)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to parse diffID %q of layer %q", cID, id)
		}
		if info, err := d.store.Info(ctx, dgst); err == nil {
			size += info.Size
		}
	}

	return size, nil
}

// Diff produces an archive of the changes between the specified
// layer and its parent layer which may be "".
func (d *Driver) Diff(id string, idMappings *idtools.IDMappings, parent string, parentMappings *idtools.IDMappings, mountLabel string) (io.ReadCloser, error) {
	if !d.isParent(id, parent) {
		return d.naiveDiff.Diff(id, idMappings, parent, parentMappings, mountLabel)
	}

	d.locker.Lock(id)
	if _, err := d.getSnapshotIDLock(id); err != nil {
		return nil, errors.Wrapf(err, "diff: snapshot doesn't registered for given id %q", id)
	}
	cID, err := d.getContentIDLock(id)
	if err != nil {
		d.locker.Unlock(id)
		// The content isn't registered by DiffApply. We cannot provide any content.
		// TODO: write it also to the content store.
		return d.naiveDiff.Diff(id, idMappings, parent, parentMappings, mountLabel)
	}
	d.locker.Unlock(id)

	diffID, err := digest.Parse(cID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse diffID %q for layer %q", cID, id)
	}
	ctx := namedCtx() // TODO: timeout?
	info, err := d.store.Info(ctx, diffID)
	if err != nil {
		// Wait for writing completion
		doneCh := make(chan content.Info)
		errCh := make(chan error)
		go func() {
			for {
				var err error

				// Check if the content is written in progress
				// TODO: Check the Expected digest here but currently containerd
				//       doesn't support getting `Expected` field by `Status`.
				if _, err = d.store.Status(ctx, id); err == nil /* && st.Expected == diffID */ {
					// writing (diffing) in progress
					time.Sleep(time.Second)
					continue
				}

				// Check if the content exists as a committed content
				if info, err = d.store.Info(ctx, diffID); err == nil {
					doneCh <- info
					return
				}

				// failed to find content
				errCh <- fmt.Errorf("data lost; layer %q: %v", id, err)
				return
			}
		}()
		select {
		case <-time.After(30 * time.Minute):
			return nil, fmt.Errorf("timed out for writing diff content of %q", id)
		case err := <-errCh:
			return nil, errors.Wrapf(err, "failed to get diff of %q", id)
		case info = <-doneCh:
		}
	}
	r, err := d.store.ReaderAt(ctx, ocispec.Descriptor{
		Digest: diffID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get diff reader of %q", id)
	}

	// Preserve the original compression method
	return ioutil.NopCloser(io.NewSectionReader(r, 0, info.Size)), nil
}

// Changes produces a list of changes between the specified layer and its
// parent layer.
func (d *Driver) Changes(id string, idMappings *idtools.IDMappings, parent string, parentIDMappings *idtools.IDMappings, mountLabel string) ([]archive.Change, error) {
	return d.naiveDiff.Changes(id, idMappings, parent, parentIDMappings, mountLabel)
}

func (d *Driver) getSnapshotIDLock(id string) (string, error) {
	var sIDs []string
	if err := d.snapshotter.Walk(namedCtx(), func(ctx context.Context, info snapshots.Info) error {
		sIDs = append(sIDs, info.Name)
		return nil
	}, fmt.Sprintf("labels.%q==%q", labelSnapshotID, id)); err != nil {
		return "", err
	}
	if len(sIDs) == 0 {
		return "", fmt.Errorf("snapshot for id %q not found", id)
	} else if len(sIDs) > 1 {
		return "", fmt.Errorf("duplicated snapshot ID: %v", sIDs)
	}

	return sIDs[0], nil
}

func (d *Driver) getContentIDLock(id string) (string, error) {
	var cIDs []string
	if err := d.snapshotter.Walk(namedCtx(), func(ctx context.Context, info snapshots.Info) error {
		if cID, ok := info.Labels[labelContentDigest]; ok {
			cIDs = append(cIDs, cID)
		}
		return nil
	}, fmt.Sprintf("labels.%q==%q", labelSnapshotID, id)); err != nil {
		return "", err
	}
	if len(cIDs) == 0 {
		return "", fmt.Errorf("content for id %q not registered", id)
	} else if len(cIDs) > 1 {
		return "", fmt.Errorf("duplicated content ID: %v", cIDs)
	}

	return cIDs[0], nil
}

func (d *Driver) isParent(id, parent string) bool {
	d.locker.Lock(id)
	defer d.locker.Unlock(id)

	key, err := d.getSnapshotIDLock(id)
	if err != nil {
		return false
	}

	pKey := ""
	if parent != "" {
		pKey, err = d.getSnapshotIDLock(parent)
		if err != nil {
			return false
		}
	}
	info, err := d.snapshotter.Stat(namedCtx(), key)
	if err != nil {
		return false
	}
	if info.Parent != pKey {
		return false
	}
	return true
}

func (d *Driver) dir(id string) string {
	return path.Join(d.home, id)
}

func (d *Driver) addLayerToStoreLock(l *Layer) error {
	data, err := ioutil.ReadFile(d.layerfile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	layers := []*Layer{}
	if err = json.Unmarshal(data, &layers); len(data) == 0 || err == nil {
		for _, layer := range layers {
			if layer.ID == l.ID {
				return fmt.Errorf("layer %q already exists", layer.ID)
			}
		}
		layers = append(layers, l)
	}

	jldata, err := json.Marshal(&layers)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(d.layerfile, jldata, 0600)
}

// NOTE: This structure has been copied from github.com/containers/storage/layers.go
// A Layer is a record of a copy-on-write layer that's stored by the lower
// level graph driver.
type Layer struct {
	// ID is either one which was specified at create-time, or a random
	// value which was generated by the library.
	ID string `json:"id"`

	// Names is an optional set of user-defined convenience values.  The
	// layer can be referred to by its ID or any of its names.  Names are
	// unique among layers.
	Names []string `json:"names,omitempty"`

	// Parent is the ID of a layer from which this layer inherits data.
	Parent string `json:"parent,omitempty"`

	// Metadata is data we keep for the convenience of the caller.  It is not
	// expected to be large, since it is kept in memory.
	Metadata string `json:"metadata,omitempty"`

	// MountLabel is an SELinux label which should be used when attempting to mount
	// the layer.
	MountLabel string `json:"mountlabel,omitempty"`

	// MountPoint is the path where the layer is mounted, or where it was most
	// recently mounted.  This can change between subsequent Unmount() and
	// Mount() calls, so the caller should consult this value after Mount()
	// succeeds to find the location of the container's root filesystem.
	MountPoint string `json:"-"`

	// MountCount is used as a reference count for the container's layer being
	// mounted at the mount point.
	MountCount int `json:"-"`

	// Created is the datestamp for when this layer was created.  Older
	// versions of the library did not track this information, so callers
	// will likely want to use the IsZero() method to verify that a value
	// is set before using it.
	Created time.Time `json:"created,omitempty"`

	// CompressedDigest is the digest of the blob that was last passed to
	// ApplyDiff() or Put(), as it was presented to us.
	CompressedDigest digest.Digest `json:"compressed-diff-digest,omitempty"`

	// CompressedSize is the length of the blob that was last passed to
	// ApplyDiff() or Put(), as it was presented to us.  If
	// CompressedDigest is not set, this should be treated as if it were an
	// uninitialized value.
	CompressedSize int64 `json:"compressed-size,omitempty"`

	// UncompressedDigest is the digest of the blob that was last passed to
	// ApplyDiff() or Put(), after we decompressed it.  Often referred to
	// as a DiffID.
	UncompressedDigest digest.Digest `json:"diff-digest,omitempty"`

	// UncompressedSize is the length of the blob that was last passed to
	// ApplyDiff() or Put(), after we decompressed it.  If
	// UncompressedDigest is not set, this should be treated as if it were
	// an uninitialized value.
	UncompressedSize int64 `json:"diff-size,omitempty"`

	// CompressionType is the type of compression which we detected on the blob
	// that was last passed to ApplyDiff() or Put().
	CompressionType archive.Compression `json:"compression,omitempty"`

	// UIDs and GIDs are lists of UIDs and GIDs used in the layer.  This
	// field is only populated (i.e., will only contain one or more
	// entries) if the layer was created using ApplyDiff() or Put().
	UIDs []uint32 `json:"uidset,omitempty"`
	GIDs []uint32 `json:"gidset,omitempty"`

	// Flags is arbitrary data about the layer.
	Flags map[string]interface{} `json:"flags,omitempty"`

	// UIDMap and GIDMap are used for setting up a layer's contents
	// for use inside of a user namespace where UID mapping is being used.
	UIDMap []idtools.IDMap `json:"uidmap,omitempty"`
	GIDMap []idtools.IDMap `json:"gidmap,omitempty"`

	// ReadOnly is true if this layer resides in a read-only layer store.
	ReadOnly bool `json:"-"`
}

type withLeaseFunc func(ctx context.Context) (context.Context, func(context.Context) error, error)

func withLeaseFuncFromContainerd(ctd *containerd.Client) func(ctx context.Context) (context.Context, func(context.Context) error, error) {
	lm := ctd.LeasesService()
	return func(ctx context.Context) (context.Context, func(context.Context) error, error) {
		if _, ok := leases.FromContext(ctx); ok {
			return ctx, func(context.Context) error {
				return nil
			}, nil
		}

		l, err := lm.Create(ctx, leases.WithRandomID(), leases.WithExpiration(24*time.Hour))
		if err != nil {
			return nil, nil, err
		}

		ctx = leases.WithLease(ctx, l.ID)
		return ctx, func(ctx context.Context) error {
			return lm.Delete(ctx, l)
		}, nil
	}
}

func namedCtx() context.Context {
	return namespaces.WithNamespace(context.Background(), namespace)
}

func uniqueKey() (string, error) {
	for i := 0; i < 5; i++ {
		key := stringid.GenerateRandomID()
		if _, err := digest.Parse(key); err == nil {
			// Key mustn't conflict with digests.
			// containerd's remote snapshotters uses digests as keys internally
			continue
		}
		return key, nil
	}
	return "", fmt.Errorf("failed to generate unique key that doesn't match digest")
}

// UpdateLayerIDMap updates ID mappings in a layer from matching the ones
// specified by toContainer to those specified by toHost.
func (d *Driver) UpdateLayerIDMap(id string, toContainer, toHost *idtools.IDMappings, mountLabel string) error {
	return fmt.Errorf("containerd driver currently doesn't support changing ID mappings")
}

// SupportsShifting tells whether the driver support shifting of the UIDs/GIDs in an userNS
func (d *Driver) SupportsShifting() bool {
	return false
}
