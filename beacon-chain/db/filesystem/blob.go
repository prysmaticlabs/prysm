package filesystem

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/verification"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/io/file"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/logging"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var (
	errIndexOutOfBounds = errors.New("blob index in file name > MaxBlobsPerBlock")
)

const (
	sszExt  = "ssz"
	partExt = "part"

	directoryPermissions = 0700
)

// NewBlobStorage creates a new instance of the BlobStorage object. Note that the implementation of BlobStorage may
// attempt to hold a file lock to guarantee exclusive control of the blob storage directory, so this should only be
// initialized once per beacon node.
func NewBlobStorage(base string) (*BlobStorage, error) {
	base = path.Clean(base)
	if err := file.MkdirAll(base); err != nil {
		return nil, err
	}
	fs := afero.NewBasePathFs(afero.NewOsFs(), base)
	return &BlobStorage{fs: fs}, nil
}

// BlobStorage is the concrete implementation of the filesystem backend for saving and retrieving BlobSidecars.
type BlobStorage struct {
	fs afero.Fs
}

// Save saves blobs given a list of sidecars.
func (bs *BlobStorage) Save(sidecar blocks.VerifiedROBlob) error {
	fname := namerForSidecar(sidecar)
	sszPath := fname.path()
	exists, err := afero.Exists(bs.fs, sszPath)
	if err != nil {
		return err
	}
	if exists {
		log.WithFields(logging.BlobFields(sidecar.ROBlob)).Debug("ignoring a duplicate blob sidecar Save attempt")
		return nil
	}

	// Serialize the ethpb.BlobSidecar to binary data using SSZ.
	sidecarData, err := sidecar.MarshalSSZ()
	if err != nil {
		return errors.Wrap(err, "failed to serialize sidecar data")
	}
	if err := bs.fs.MkdirAll(fname.dir(), directoryPermissions); err != nil {
		return err
	}
	partPath := fname.partPath()
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

// Get retrieves a single BlobSidecar by its root and index.
// Since BlobStorage only writes blobs that have undergone full verification, the return
// value is always a VerifiedROBlob.
func (bs *BlobStorage) Get(root [32]byte, idx uint64) (blocks.VerifiedROBlob, error) {
	expected := blobNamer{root: root, index: idx}
	encoded, err := afero.ReadFile(bs.fs, expected.path())
	var v blocks.VerifiedROBlob
	if err != nil {
		return v, err
	}
	s := &ethpb.BlobSidecar{}
	if err := s.UnmarshalSSZ(encoded); err != nil {
		return v, err
	}
	ro, err := blocks.NewROBlobWithRoot(s, root)
	if err != nil {
		return blocks.VerifiedROBlob{}, err
	}
	return verification.BlobSidecarNoop(ro)
}

// Indices generates a bitmap representing which BlobSidecar.Index values are present on disk for a given root.
// This value can be compared to the commitments observed in a block to determine which indices need to be found
// on the network to confirm data availability.
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
		if u >= fieldparams.MaxBlobsPerBlock {
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

func (p blobNamer) partPath() string {
	return p.fname(partExt)
}

func (p blobNamer) path() string {
	return p.fname(sszExt)
}
