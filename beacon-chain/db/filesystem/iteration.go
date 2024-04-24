package filesystem

import (
	"encoding/binary"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/spf13/afero"
)

func listDir(fs afero.Fs, dir string) ([]string, error) {
	top, err := fs.Open(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open directory descriptor")
	}
	defer func() {
		if err := top.Close(); err != nil {
			log.WithError(err).Errorf("Could not close file %s", dir)
		}
	}()
	// re the -1 param: "If n <= 0, Readdirnames returns all the names from the directory in a single slice"
	dirs, err := top.Readdirnames(-1)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read directory listing")
	}
	return dirs, nil
}

type layoutLevel struct {
	namerUpdater namerUpdater
	filter       func(string) bool
}

type namerUpdater func(blobNamer, string) (blobNamer, error)

type namerIterator struct {
	fs      afero.Fs
	path    string
	child   *namerIterator
	namer   blobNamer
	levels  []layoutLevel
	entries []string
	offset  int
}

func (iter *namerIterator) next() (blobNamer, error) {
	if iter.child != nil {
		next, err := iter.child.next()
		if err == nil {
			return next, nil
		}
		if err != io.EOF {
			return blobNamer{}, err
		}
	}
	return iter.advanceChild()
}

func (iter *namerIterator) advanceChild() (blobNamer, error) {
	for i := iter.offset; i < len(iter.entries); i++ {
		iter.offset = i
		nextPath := filepath.Join(iter.path, iter.entries[iter.offset])
		nextLevel := iter.levels[0]
		if !nextLevel.filter(nextPath) {
			continue
		}
		namer, err := nextLevel.namerUpdater(iter.namer, nextPath)
		if err != nil {
			return namer, err
		}
		// if we're at the leaf level, we can return the updated namer.
		if len(iter.levels) == 1 {
			iter.offset += 1
			return namer, nil
		}

		entries, err := listDir(iter.fs, nextPath)
		if err != nil {
			return blobNamer{}, err
		}
		if len(entries) == 0 {
			return blobNamer{}, io.EOF
		}
		iter.child = &namerIterator{
			fs:      iter.fs,
			path:    nextPath,
			namer:   namer,
			levels:  iter.levels[1:],
			entries: entries,
		}
		iter.offset += 1
		return iter.child.next()
	}

	return blobNamer{}, io.EOF
}

func noopNamerUpdater(namer blobNamer, dir string) (blobNamer, error) {
	return namer, nil
}

func epochNamerUpdater(namer blobNamer, dir string) (blobNamer, error) {
	epoch, err := epochFromPath(dir)
	if err != nil {
		return namer, err
	}
	namer.epoch = epoch
	return namer, nil
}

func rootNamerUpdater(namer blobNamer, dir string) (blobNamer, error) {
	root, err := rootFromPath(dir)
	if err != nil {
		return namer, err
	}
	namer.root = root
	return namer, nil
}

func indexNamerUpdater(namer blobNamer, fname string) (blobNamer, error) {
	idx, err := idxFromPath(fname)
	if err != nil {
		return namer, err
	}
	namer.index = idx
	return namer, nil
}

type readSlotOncePerRoot struct {
	fs       afero.Fs
	lastRoot [32]byte
	epoch    primitives.Epoch
}

func (l *readSlotOncePerRoot) namerUpdater(namer blobNamer, fname string) (blobNamer, error) {
	namer, err := indexNamerUpdater(namer, fname)
	if err != nil {
		return namer, err
	}
	if namer.root != l.lastRoot {
		slot, err := slotFromFile(fname, l.fs)
		if err != nil {
			return namer, err
		}
		l.lastRoot = namer.root
		l.epoch = slots.ToEpoch(slot)
	}
	namer.epoch = l.epoch
	return namer, nil
}

func epochFromPath(p string) (primitives.Epoch, error) {
	subdir := filepath.Base(p)
	epoch, err := strconv.ParseUint(subdir, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(errInvalidDirectoryLayout,
			"failed to decode epoch as uint, err=%s, dir=%s", err.Error(), p)
	}
	return primitives.Epoch(epoch), nil
}

func rootFromPath(p string) ([32]byte, error) {
	subdir := filepath.Base(p)
	root, err := stringToRoot(subdir)
	if err != nil {
		return root, errors.Wrapf(err, "invalid directory, could not parse subdir as root %s", p)
	}
	return root, nil
}

func idxFromPath(p string) (uint64, error) {
	p = path.Base(p)

	if !isSszFile(p) {
		return 0, errors.Wrap(errNotBlobSSZ, "does not have .ssz extension")
	}
	parts := strings.Split(p, ".")
	if len(parts) != 2 {
		return 0, errors.Wrap(errNotBlobSSZ, "unexpected filename structure (want <index>.ssz)")
	}
	return strconv.ParseUint(parts[0], 10, 64)
}

// Read slot from marshaled BlobSidecar data in the given file. See slotFromBlob for details.
func slotFromFile(name string, fs afero.Fs) (primitives.Slot, error) {
	f, err := fs.Open(name)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.WithError(err).Errorf("Could not close blob file")
		}
	}()
	return slotFromBlob(f)
}

// slotFromBlob reads the ssz data of a file at the specified offset (8 + 131072 + 48 + 48 = 131176 bytes),
// which is calculated based on the size of the BlobSidecar struct and is based on the size of the fields
// preceding the slot information within SignedBeaconBlockHeader.
func slotFromBlob(at io.ReaderAt) (primitives.Slot, error) {
	b := make([]byte, 8)
	_, err := at.ReadAt(b, 131176)
	if err != nil {
		return 0, err
	}
	rawSlot := binary.LittleEndian.Uint64(b)
	return primitives.Slot(rawSlot), nil
}

func filterNoop(_ string) bool {
	return true
}

func isRootDir(p string) bool {
	dir := filepath.Base(p)
	return len(dir) == rootStringLen && strings.HasPrefix(dir, "0x")
}

func isSszFile(s string) bool {
	return filepath.Ext(s) == "."+sszExt
}

func rootToString(root [32]byte) string {
	return fmt.Sprintf("%#x", root)
}

func stringToRoot(str string) ([32]byte, error) {
	if len(str) != rootStringLen {
		return [32]byte{}, errors.Wrapf(errInvalidRootString, "incorrect len for input=%s", str)
	}
	slice, err := hexutil.Decode(str)
	if err != nil {
		return [32]byte{}, errors.Wrapf(errInvalidRootString, "input=%s", str)
	}
	return bytesutil.ToBytes32(slice), nil
}
