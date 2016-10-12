package cri

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Sirupsen/logrus"
	istorage "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/storage/storage"
	"golang.org/x/net/context"
)

var (
	// ErrInvalidPodName is returned when a pod name specified to a
	// function call is found to be invalid (most often, because it's
	// empty).
	ErrInvalidPodName = errors.New("invalid pod name")
	// ErrInvalidImageName is returned when an image name specified to a
	// function call is found to be invalid (most often, because it's
	// empty).
	ErrInvalidImageName = errors.New("invalid image name")
	// ErrInvalidContainerName is returned when a container name specified
	// to a function call is found to be invalid (most often, because it's
	// empty).
	ErrInvalidContainerName = errors.New("invalid container name")
	// ErrInvalidSandboxID is returned when a sandbox ID specified to a
	// function call is found to be invalid (because it's either
	// empty or doesn't match a valid sandbox).
	ErrInvalidSandboxID = errors.New("invalid sandbox ID")
	// ErrInvalidContainerID is returned when a container ID specified to a
	// function call is found to be invalid (because it's either
	// empty or doesn't match a valid container).
	ErrInvalidContainerID = errors.New("invalid container ID")
)

type runtimeService struct {
	image ImageServer
}

// ContainerInfo wraps a subset of information about a container: its ID and
// the locations of its nonvolatile and volatile per-container directories.
type ContainerInfo struct {
	ID     string
	Dir    string
	RunDir string
}

// RuntimeServer wraps up various CRI-related activities into a reusable
// implementation.
type RuntimeServer interface {
	// CreatePodSandbox creates a pod infrastructure container, using the
	// specified PodID for the infrastructure container's ID.  In the CRI
	// view of things, a sandbox is distinct from its containers, including
	// its infrastructure container, but at this level the sandbox is
	// essentially the same as its infrastructure container, with a
	// container's membership in a pod being signified by it listing the
	// same pod ID in its metadata that the pod's other members do, and
	// with the pod's infrastructure container having the same value for
	// both its pod's ID and its container ID.
	// The metadata structure should have its PodName and PodID specified.
	// The metadata structure should have at least either its ImageName or ImageID specified.
	// The metadata structure should have its ContainerName and MetadataName specified.
	CreatePodSandbox(ctx context.Context, metadata RuntimeContainerMetadata) (ContainerInfo, error)
	// RemovePodSandbox deletes a pod sandbox's infrastructure container.
	// The CRI expects that a sandbox can't be removed unless its only
	// container is its infrastructure container, but we don't enforce that
	// here, since we're just keeping track of it for higher level APIs.
	RemovePodSandbox(idOrName string) error

	// GetContainerMetadata returns the metadata we've stored for a container.
	GetContainerMetadata(idOrName string) (RuntimeContainerMetadata, error)
	// SetContainerMetadata updates the metadata we've stored for a container.
	SetContainerMetadata(idOrName string, metadata RuntimeContainerMetadata) error

	// CreateContainer creates a container with the specified ID.
	// The metadata structure should have its PodName and PodID specified.
	// The metadata structure should have at least either its ImageName or ImageID specified.
	// The metadata structure should have its ContainerName and MetadataName specified.
	CreateContainer(ctx context.Context, metadata RuntimeContainerMetadata, containerID string) (ContainerInfo, error)
	// DeleteContainer deletes a container, unmounting it first if need be.
	DeleteContainer(idOrName string) error

	// StartContainer makes sure a container's filesystem is mounted, and
	// returns the location of its root filesystem, which is not guaranteed
	// by lower-level drivers to never change.
	StartContainer(idOrName string) (string, error)
	// StopContainer attempts to unmount a container's root filesystem,
	// freeing up any kernel resources which may be limited.
	StopContainer(idOrName string) error

	// GetWorkDir returns the path of a nonvolatile directory on the
	// filesystem (somewhere under the Store's Root directory) which can be
	// used to store arbitrary data that is specific to the container.  It
	// will be removed automatically when the container is deleted.
	GetWorkDir(id string) (string, error)
	// GetRunDir returns the path of a volatile directory (does not survive
	// the host rebooting, somewhere under the Store's RunRoot directory)
	// on the filesystem which can be used to store arbitrary data that is
	// specific to the container.  It will be removed automatically when
	// the container is deleted.
	GetRunDir(id string) (string, error)
}

// RuntimeContainerMetadata is the structure that we encode as JSON and store
// in the metadata field of storage.Container objects.  It is used for
// specifying attributes of pod sandboxes and containers when they are being
// created, and allows a container's MountLabel, and possibly other values, to
// be modified in one read/write cycle via calls to
// RuntimeServer.GetContainerMetadata, RuntimeContainerMetadata.SetMountLabel,
// and RuntimeServer.SetContainerMetadata.
type RuntimeContainerMetadata struct {
	// Pod is true if this is the pod's infrastructure container.
	Pod bool `json:"pod,omitempty"` // Applicable to both PodSandboxes and Containers
	// The pod's name and ID, kept for use by upper layers in determining
	// which containers belong to which pods.
	PodName string `json:"pod-name"` // Applicable to both PodSandboxes and Containers, mandatory
	PodID   string `json:"pod-id"`   // Applicable to both PodSandboxes and Containers, mandatory
	// The provided name and the ID of the image that was used to
	// instantiate the container.
	ImageName string `json:"image-name"` // Applicable to both PodSandboxes and Containers
	ImageID   string `json:"image-id"`   // Applicable to both PodSandboxes and Containers
	// The container's name, which for an infrastructure container is usually PodName + "-infra".
	ContainerName string `json:"name"` // Applicable to both PodSandboxes and Containers, mandatory
	// The name as originally specified in PodSandbox or Container CRI metadata.
	MetadataName string `json:"metadata-name"`        // Applicable to both PodSandboxes and Containers, mandatory
	UID          string `json:"uid,omitempty"`        // Only applicable to pods
	Namespace    string `json:"namespace,omitempty"`  // Only applicable to pods
	Attempt      uint32 `json:"attempt,omitempty"`    // Applicable to both PodSandboxes and Containers
	CreatedAt    int64  `json:"created-at"`           // Applicable to both PodSandboxes and Containers
	MountLabel   string `json:"mountlabel,omitempty"` // Applicable to both PodSandboxes and Containers
}

// SetMountLabel updates the mount label held by a RuntimeContainerMetadata
// object.
func (metadata *RuntimeContainerMetadata) SetMountLabel(mountLabel string) {
	metadata.MountLabel = mountLabel
}

func (r *runtimeService) createContainerOrPodSandbox(ctx context.Context, metadata RuntimeContainerMetadata, containerID string) (ContainerInfo, error) {
	if metadata.PodName == "" || metadata.PodID == "" {
		return ContainerInfo{}, ErrInvalidPodName
	}
	if metadata.ContainerName == "" {
		return ContainerInfo{}, ErrInvalidContainerName
	}
	if metadata.MetadataName == "" {
		metadata.MetadataName = metadata.ContainerName
	}
	if metadata.CreatedAt == 0 {
		metadata.CreatedAt = time.Now().Unix()
	}

	// Check if we have the specified image.
	ref, err := istorage.Transport.ParseStoreReference(r.image.GetStore(), metadata.ImageName)
	if err != nil {
		// Maybe it's some other transport's copy of the image?
		otherRef, err2 := transports.ParseImageName(metadata.ImageName)
		if err2 == nil && otherRef.DockerReference() != nil {
			ref, err = istorage.Transport.ParseStoreReference(r.image.GetStore(), otherRef.DockerReference().FullName())
		}
		if err != nil {
			return ContainerInfo{}, err
		}
	}
	img, err := istorage.Transport.GetStoreImage(r.image.GetStore(), ref)
	if img == nil && err == storage.ErrImageUnknown && metadata.ImageID != "" {
		ref, err = istorage.Transport.ParseStoreReference(r.image.GetStore(), "@"+metadata.ImageID)
		if err != nil {
			return ContainerInfo{}, err
		}
		img, err = istorage.Transport.GetStoreImage(r.image.GetStore(), ref)
	}
	if err != nil && err != storage.ErrImageUnknown {
		return ContainerInfo{}, err
	}

	// Pull the image down if we don't already have it.
	if err == storage.ErrImageUnknown {
		image := metadata.ImageID
		if metadata.ImageName != "" {
			image = metadata.ImageName
		}
		if image == "" {
			return ContainerInfo{}, ErrInvalidImageName
		}
		logrus.Debugf("couldn't find image %q, retrieving it", image)
		ref, err := r.image.PullImage(ctx, image)
		if err != nil {
			return ContainerInfo{}, err
		}
		img, err = istorage.Transport.GetStoreImage(r.image.GetStore(), ref)
		if err != nil {
			return ContainerInfo{}, err
		}
		logrus.Debugf("successfully pulled image %q", image)
	}

	// Fill in the image name and ID.
	if metadata.ImageName == "" && len(img.Names) > 0 {
		metadata.ImageName = img.Names[0]
	}
	metadata.ImageID = img.ID

	// For now, we have a default cleanup function.
	cleanup := func(container *storage.Container) {
		if err2 := r.image.GetStore().DeleteContainer(container.ID); err2 != nil {
			if metadata.Pod {
				logrus.Infof("%v deleting partially-created pod sandbox %q", err2, container.ID)
			} else {
				logrus.Infof("%v deleting partially-created container %q", err2, container.ID)
			}
			return
		}
		logrus.Infof("deleted partially-created container %q", container.ID)
	}

	// Build metadata to store with the container.
	mdata, err := json.Marshal(&metadata)
	if err != nil {
		return ContainerInfo{}, err
	}

	// Build the container.
	names := []string{metadata.ContainerName}
	if metadata.Pod {
		names = append(names, metadata.PodName)
	}
	container, err := r.image.GetStore().CreateContainer(containerID, names, img.ID, "", string(mdata), nil)
	if err != nil {
		if metadata.Pod {
			logrus.Debugf("failed to create pod sandbox %s(%s): %v", metadata.PodName, metadata.PodID, err)
		} else {
			logrus.Debugf("failed to create container %s(%s): %v", metadata.ContainerName, containerID, err)
		}
		return ContainerInfo{}, err
	}
	if metadata.Pod {
		logrus.Debugf("created pod sandbox %q", container.ID)
	} else {
		logrus.Debugf("created container %q", container.ID)
	}

	// Add a name to the container's layer so that it's easier to follow
	// what's going on if we're just looking at the storage-eye view of things.
	layerName := metadata.ContainerName + "-layer"
	names, err = r.image.GetStore().GetNames(container.LayerID)
	if err != nil {
		cleanup(container)
		return ContainerInfo{}, err
	}
	names = append(names, layerName)
	err = r.image.GetStore().SetNames(container.LayerID, names)
	if err != nil {
		cleanup(container)
		return ContainerInfo{}, err
	}

	// Find out where the container work directories are, so that we can return them.
	containerDir, err := r.image.GetStore().GetContainerDirectory(container.ID)
	if err != nil {
		cleanup(container)
		return ContainerInfo{}, err
	}
	if metadata.Pod {
		logrus.Debugf("pod sandbox %q has work directory %q", container.ID, containerDir)
	} else {
		logrus.Debugf("container %q has work directory %q", container.ID, containerDir)
	}

	containerRunDir, err := r.image.GetStore().GetContainerRunDirectory(container.ID)
	if err != nil {
		cleanup(container)
		return ContainerInfo{}, err
	}
	if metadata.Pod {
		logrus.Debugf("pod sandbox %q has run directory %q", container.ID, containerRunDir)
	} else {
		logrus.Debugf("container %q has run directory %q", container.ID, containerRunDir)
	}

	return ContainerInfo{
		ID:     container.ID,
		Dir:    containerDir,
		RunDir: containerRunDir,
	}, nil
}

func (r *runtimeService) CreatePodSandbox(ctx context.Context, metadata RuntimeContainerMetadata) (ContainerInfo, error) {
	metadata.Pod = true
	return r.createContainerOrPodSandbox(ctx, metadata, metadata.PodID)
}

func (r *runtimeService) CreateContainer(ctx context.Context, metadata RuntimeContainerMetadata, containerID string) (ContainerInfo, error) {
	metadata.Pod = false
	return r.createContainerOrPodSandbox(ctx, metadata, containerID)
}

func (r *runtimeService) RemovePodSandbox(idOrName string) error {
	container, err := r.image.GetStore().GetContainer(idOrName)
	if err != nil {
		if err == storage.ErrContainerUnknown {
			return ErrInvalidSandboxID
		}
		return err
	}
	err = r.image.GetStore().DeleteContainer(container.ID)
	if err != nil {
		logrus.Debugf("failed to delete pod sandbox %q: %v", container.ID, err)
		return err
	}
	return nil
}

func (r *runtimeService) DeleteContainer(idOrName string) error {
	container, err := r.image.GetStore().GetContainer(idOrName)
	if err != nil {
		if err == storage.ErrContainerUnknown {
			return ErrInvalidContainerID
		}
		return err
	}
	err = r.image.GetStore().DeleteContainer(container.ID)
	if err != nil {
		logrus.Debugf("failed to delete container %q: %v", container.ID, err)
		return err
	}
	return nil
}

func (r *runtimeService) SetContainerMetadata(idOrName string, metadata RuntimeContainerMetadata) error {
	mdata, err := json.Marshal(&metadata)
	if err != nil {
		logrus.Debugf("failed to encode metadata for %q: %v", idOrName, err)
		return err
	}
	return r.image.GetStore().SetMetadata(idOrName, string(mdata))
}

func (r *runtimeService) GetContainerMetadata(idOrName string) (RuntimeContainerMetadata, error) {
	metadata := RuntimeContainerMetadata{}
	mdata, err := r.image.GetStore().GetMetadata(idOrName)
	if err != nil {
		return RuntimeContainerMetadata{}, err
	}
	if err = json.Unmarshal([]byte(mdata), &metadata); err != nil {
		return RuntimeContainerMetadata{}, err
	}
	return metadata, err
}

func (r *runtimeService) StartContainer(idOrName string) (string, error) {
	container, err := r.image.GetStore().GetContainer(idOrName)
	if err != nil {
		if err == storage.ErrContainerUnknown {
			return "", ErrInvalidContainerID
		}
		return "", err
	}
	metadata := RuntimeContainerMetadata{}
	if err = json.Unmarshal([]byte(container.Metadata), &metadata); err != nil {
		return "", err
	}
	mountPoint, err := r.image.GetStore().Mount(container.ID, metadata.MountLabel)
	if err != nil {
		logrus.Debugf("failed to mount container %q: %v", container.ID, err)
		return "", err
	}
	logrus.Debugf("mounted container %q at %q", container.ID, mountPoint)
	return mountPoint, nil
}

func (r *runtimeService) StopContainer(idOrName string) error {
	container, err := r.image.GetStore().GetContainer(idOrName)
	if err != nil {
		if err == storage.ErrContainerUnknown {
			return ErrInvalidContainerID
		}
		return err
	}
	err = r.image.GetStore().Unmount(container.ID)
	if err != nil {
		logrus.Debugf("failed to unmount container %q: %v", container.ID, err)
		return err
	}
	logrus.Debugf("unmounted container %q", container.ID)
	return nil
}

func (r *runtimeService) GetWorkDir(id string) (string, error) {
	container, err := r.image.GetStore().GetContainer(id)
	if err != nil {
		if err == storage.ErrContainerUnknown {
			return "", ErrInvalidContainerID
		}
		return "", err
	}
	return r.image.GetStore().GetContainerDirectory(container.ID)
}

func (r *runtimeService) GetRunDir(id string) (string, error) {
	container, err := r.image.GetStore().GetContainer(id)
	if err != nil {
		if err == storage.ErrContainerUnknown {
			return "", ErrInvalidContainerID
		}
		return "", err
	}
	return r.image.GetStore().GetContainerRunDirectory(container.ID)
}

// GetRuntimeService returns a RuntimeServer that uses the passed-in image
// service to pull and manage images, and its store to manage containers based
// on those images.
func GetRuntimeService(image ImageServer) RuntimeServer {
	return &runtimeService{
		image: image,
	}
}
