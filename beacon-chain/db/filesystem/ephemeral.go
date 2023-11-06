package filesystem

import "github.com/spf13/afero"

func NewEphemeralBlobStorage() *BlobStorage {
	return &BlobStorage{fs: afero.NewMemMapFs()}
}

func NewEphemeralBlobStorageWithFs() (afero.Fs, *BlobStorage) {
	fs := afero.NewMemMapFs()
	return fs, &BlobStorage{fs: fs}
}
