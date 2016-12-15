package cri

import (
	"github.com/containers/image/copy"
	"github.com/containers/image/signature"
	istorage "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	"github.com/containers/storage/storage"
	"golang.org/x/net/context"
)

// ImageResult wraps a subset of information about an image: its ID, its names,
// and the size, if known, or nil if it isn't.
type ImageResult struct {
	ID    string
	Names []string
	Size  *uint64
}

type imageService struct {
	store            storage.Store
	defaultTransport string
}

// ImageServer wraps up various CRI-related activities into a reusable
// implementation.
type ImageServer interface {
	// ListImages returns list of all images which match the filter.
	ListImages(ctx context.Context, filter string) ([]ImageResult, error)
	// ImageStatus returns status of an image which matches the filter.
	ImageStatus(ctx context.Context, filter string) (*ImageResult, error)
	// PullImageUsingContexts imports an image from the specified location.
	PullImageUsingContexts(ctx context.Context, imageName string, policyContext *signature.PolicyContext, options *copy.Options) (types.ImageReference, error)
	// PullImage imports an image from the specified location and default settings.
	PullImage(ctx context.Context, imageName string) (types.ImageReference, error)
	// RemoveImage deletes the specified image.
	RemoveImage(ctx context.Context, imageName string) error
	GetStore() storage.Store
}

func (svc *imageService) ListImages(ctx context.Context, filter string) ([]ImageResult, error) {
	results := []ImageResult{}
	if filter != "" {
		if image, err := svc.store.GetImage(filter); err == nil {
			results = append(results, ImageResult{
				ID:    image.ID,
				Names: image.Names,
			})
		}
	} else {
		images, err := svc.store.Images()
		if err != nil {
			return nil, err
		}
		for _, image := range images {
			results = append(results, ImageResult{
				ID:    image.ID,
				Names: image.Names,
			})
		}
	}
	return results, nil
}

func (svc *imageService) ImageStatus(ctx context.Context, nameOrID string) (*ImageResult, error) {
	ref, err := transports.ParseImageName(nameOrID)
	if err != nil {
		ref2, err2 := istorage.Transport.ParseStoreReference(svc.store, "@"+nameOrID)
		if err2 != nil {
			ref3, err3 := istorage.Transport.ParseStoreReference(svc.store, nameOrID)
			if err3 != nil {
				return nil, err
			}
			ref2 = ref3
		}
		ref = ref2
	}
	image, err := istorage.Transport.GetStoreImage(svc.store, ref)
	if err != nil {
		return nil, err
	}

	systemContext := types.SystemContext{}
	img, err := ref.NewImage(&systemContext)
	if err != nil {
		return nil, err
	}
	size := imageSize(img)
	img.Close()

	return &ImageResult{
		ID:    image.ID,
		Names: image.Names,
		Size:  size,
	}, nil
}

func imageSize(img types.Image) *uint64 {
	if sizer, ok := img.(istorage.Sizer); ok {
		sum, err := sizer.Size()
		if err != nil || sum < 0 {
			return nil
		}
		usum := uint64(sum)
		return &usum
	}
	return nil
}

func (svc *imageService) PullImageUsingContexts(ctx context.Context, imageName string, policyContext *signature.PolicyContext, options *copy.Options) (types.ImageReference, error) {
	if imageName == "" {
		return nil, storage.ErrNotAnImage
	}
	srcRef, err := transports.ParseImageName(imageName)
	if err != nil {
		srcRef2, err2 := transports.ParseImageName(svc.defaultTransport + imageName)
		if err2 != nil {
			return nil, err
		}
		srcRef = srcRef2
	}
	dest := imageName
	if srcRef.DockerReference() != nil {
		dest = srcRef.DockerReference().FullName()
	}
	destRef, err := istorage.Transport.ParseStoreReference(svc.store, dest)
	if err != nil {
		return nil, err
	}
	err = copy.Image(policyContext, destRef, srcRef, options)
	if err != nil {
		return nil, err
	}
	// Go find the image, and attach the requested name to it, so that we
	// can more easily find it later, even if the destination reference
	// looks different.
	destImage, err := istorage.Transport.GetStoreImage(svc.store, destRef)
	if err != nil {
		return nil, err
	}
	names := append(destImage.Names, imageName, dest)
	err = svc.store.SetNames(destImage.ID, names)
	if err != nil {
		return nil, err
	}
	return destRef, nil
}

func (svc *imageService) PullImage(ctx context.Context, imageName string) (types.ImageReference, error) {
	systemContext := types.SystemContext{}
	policy, err := signature.DefaultPolicy(&systemContext)
	if err != nil {
		return nil, err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, err
	}
	options := copy.Options{}
	ref, err := svc.PullImageUsingContexts(ctx, imageName, policyContext, &options)
	if err != nil {
		return nil, err
	}
	return ref, nil
}

func (svc *imageService) RemoveImage(ctx context.Context, nameOrID string) error {
	ref, err := transports.ParseImageName(nameOrID)
	if err != nil {
		ref2, err2 := istorage.Transport.ParseStoreReference(svc.store, "@"+nameOrID)
		if err2 != nil {
			ref3, err3 := istorage.Transport.ParseStoreReference(svc.store, nameOrID)
			if err3 != nil {
				return err
			}
			ref2 = ref3
		}
		ref = ref2
	}
	systemContext := &types.SystemContext{}
	return ref.DeleteImage(systemContext)
}

func (svc *imageService) GetStore() storage.Store {
	return svc.store
}

// GetImageService returns an ImageServer that uses the passed-in store, and
// which will prepend the passed-in defaultTransport value to an image name if
// a name that's passed to its PullImage() method can't be resolved to an image
// in the store and can't be resolved to a source on its own.
func GetImageService(store storage.Store, defaultTransport string) (ImageServer, error) {
	if store == nil {
		var err error
		store, err = storage.GetStore(storage.DefaultStoreOptions)
		if err != nil {
			return nil, err
		}
	}
	return &imageService{
		store:            store,
		defaultTransport: defaultTransport,
	}, nil
}
