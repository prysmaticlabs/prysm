package filesystem

import "github.com/spf13/afero"

// NewEphemeralBlobStorage should only be used for tests.
// The instance of BlobStorage returned is backed by an in-memory virtual filesystem,
// improving test performance and simplifying cleanup.
func NewEphemeralBlobStorage() *BlobStorage {
	return &BlobStorage{fs: afero.NewMemMapFs()}
}

// NewEphemeralBlobStorageWithFs can be used by tests that want access to the virtual filesystem
// in order to interact with it outside the parameters of the BlobStorage api.
func NewEphemeralBlobStorageWithFs() (afero.Fs, *BlobStorage) {
	fs := afero.NewMemMapFs()
	return fs, &BlobStorage{fs: fs}
}
