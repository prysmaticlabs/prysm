package filesystem

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/spf13/afero"
)

var (
	ErrBlobRetrieval    = errors.New("error while trying to read BlobSidecar from filesystem")
	errIndexOutOfBounds = errors.New("blob index in file name > MaxBlobsPerBlock")
)

const (
	sszExt       = "ssz"
	partExt      = "part"
	blobLockPath = "blob.lock"
)

func NewBlobStorage(ctx context.Context, base string) (*BlobStorage, error) {
	base = path.Clean(base)
	fs := afero.NewBasePathFs(afero.NewOsFs(), base)
	return &BlobStorage{fs: fs}, nil
}

type BlobStorage struct {
	fs afero.Fs
}

// Save saves blobs given a list of sidecars.
func (bs *BlobStorage) Save(sidecar blocks.VerifiedROBlob) error {
	fname := namerForSidecar(sidecar)
	sszPath := fname.ssz()
	exists, err := afero.Exists(bs.fs, sszPath)
	if err != nil {
		return err
	}
	if exists {
		// TODO: should it be an error to save a blob that already exists?
		return nil
	}

	// Serialize the ethpb.BlobSidecar to binary data using SSZ.
	sidecarData, err := ssz.MarshalSSZ(sidecar)
	if err != nil {
		return errors.Wrap(err, "failed to serialize sidecar data")
	}
	partPath := fname.partial()
	// Create a partial file and write the serialized data to it.
	partialFile, err := bs.fs.Create(partPath)
	if err != nil {
		return errors.Wrap(err, "failed to create partial file")
	}

	_, err = partialFile.Write(sidecarData)
	if err != nil {
		closeErr := partialFile.Close()
		if closeErr != nil {
			return closeErr
		}
		return errors.Wrap(err, "failed to write to partial file")
	}
	err = partialFile.Close()
	if err != nil {
		return err
	}

	// Atomically rename the partial file to its final name.
	err = bs.fs.Rename(partPath, sszPath)
	if err != nil {
		return errors.Wrap(err, "failed to rename partial file to final name")
	}
	return nil
}

func (bs *BlobStorage) Get(root [32]byte, idx uint64) (blocks.VerifiedROBlob, error) {
	expected := blobNamer{root: root, index: idx}
	encoded, err := afero.ReadFile(bs.fs, expected.ssz())
	var v blocks.VerifiedROBlob
	if err != nil {
		return v, err
	}
	s := &ethpb.BlobSidecar{}
	if err := s.UnmarshalSSZ(encoded); err != nil {
		return v, err
	}
	return blocks.NewVerifiedBlobWithRoot(s, root)
}

func (bs *BlobStorage) Indices(root [32]byte) ([fieldparams.MaxBlobsPerBlock]bool, error) {
	var mask [fieldparams.MaxBlobsPerBlock]bool
	rootDir := blobNamer{root: root}.dir()
	entries, err := afero.ReadDir(bs.fs, rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return mask, nil
		}
		return mask, err
	}
	for i := range entries {
		if entries[i].IsDir() {
			continue
		}
		name := entries[i].Name()
		if !strings.HasSuffix(name, sszExt) {
			continue
		}
		parts := strings.Split(name, ".")
		if len(parts) != 2 {
			continue
		}
		u, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return mask, errors.Wrapf(err, "unexpected directory entry breaks listing, %s", parts[0])
		}
		if u > fieldparams.MaxBlobsPerBlock {
			return mask, errIndexOutOfBounds
		}
		mask[u] = true
	}
	return mask, nil
}

type blobNamer struct {
	root  [32]byte
	index uint64
}

func namerForSidecar(sc blocks.VerifiedROBlob) blobNamer {
	return blobNamer{root: sc.BlockRoot(), index: sc.Index}
}

func (p blobNamer) dir() string {
	return fmt.Sprintf("%#x", p.root)
}

func (p blobNamer) fname(ext string) string {
	return path.Join(p.dir(), fmt.Sprintf("%d.%s", p.index, ext))
}

func (p blobNamer) partial() string {
	return p.fname(partExt)
}

func (p blobNamer) ssz() string {
	return p.fname(sszExt)
}
