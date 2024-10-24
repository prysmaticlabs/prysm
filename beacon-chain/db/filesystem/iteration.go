package filesystem

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var errIdentFailure = errors.New("failed to determine blob metadata, ignoring all sub-path.")

type identificationError struct {
	err   error
	path  string
	ident blobIdent
}

func (ide *identificationError) Error() string {
	return fmt.Sprintf("%s path=%s, err=%s", errIdentFailure.Error(), ide.path, ide.err.Error())
}

func (ide *identificationError) Unwrap() error {
	return ide.err
}

func (*identificationError) Is(err error) bool {
	return err == errIdentFailure
}

func (ide *identificationError) LogFields() logrus.Fields {
	fields := ide.ident.logFields()
	fields["path"] = ide.path
	return fields
}

func newIdentificationError(path string, ident blobIdent, err error) *identificationError {
	return &identificationError{path: path, ident: ident, err: err}
}

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

// identPopulator is a function that sets values in the blobIdent for a given level of the filesystem layout.
type identPopulator func(blobIdent, string) (blobIdent, error)

type layoutLevel struct {
	populateIdent identPopulator
	filter        func(string) bool
}

// identIterator moves through the filesystem in order to yield blobIdents.
// layoutLevels (in the 'levels' field) allows a filesystem layout to control how the
// layout is traversed. A layoutLevel can filter out entries from the directory listing
// via the filter function, and populate fields in the blobIdent via the populateIdent function.
// The blobIdent is populated from an empty value at the root, accumulating values for its fields at each level.
// The fully populated blobIdent is returned when the iterator reaches the leaf level.
type identIterator struct {
	fs      afero.Fs
	path    string
	child   *identIterator
	ident   blobIdent
	levels  []layoutLevel
	entries []string
	offset  int
	eof     bool
}

// next is the only method that a user of the identIterator needs to call.
// identIterator will blobIdents in a breadth-first fashion,
// returning an empty blobIdent and io.EOF once all branches have been traversed.
func (iter *identIterator) next() (blobIdent, error) {
	if iter.eof {
		return blobIdent{}, io.EOF
	}
	if iter.child != nil {
		next, err := iter.child.next()
		if err == nil {
			return next, nil
		}
		if err != io.EOF {
			return blobIdent{}, err
		}
	}
	return iter.advanceChild()
}

// advanceChild is used to move to the next directory at each level of the tree, either when
// the nodes are first being initialized at a level, or when a sub-branch has been exhausted.
func (iter *identIterator) advanceChild() (blobIdent, error) {
	defer func() {
		iter.offset += 1
	}()
	for i := iter.offset; i < len(iter.entries); i++ {
		iter.offset = i
		nextPath := filepath.Join(iter.path, iter.entries[iter.offset])
		nextLevel := iter.levels[0]
		if !nextLevel.filter(nextPath) {
			continue
		}
		ident, err := nextLevel.populateIdent(iter.ident, nextPath)
		if err != nil {
			return ident, newIdentificationError(nextPath, ident, err)
		}
		// if we're at the leaf level, we can return the updated ident.
		if len(iter.levels) == 1 {
			return ident, nil
		}

		entries, err := listDir(iter.fs, nextPath)
		if err != nil {
			return blobIdent{}, err
		}
		if len(entries) == 0 {
			continue
		}
		iter.child = &identIterator{
			fs:      iter.fs,
			path:    nextPath,
			ident:   ident,
			levels:  iter.levels[1:],
			entries: entries,
		}
		return iter.child.next()
	}

	return blobIdent{}, io.EOF
}

func populateNoop(namer blobIdent, _ string) (blobIdent, error) {
	return namer, nil
}

func populateRoot(namer blobIdent, dir string) (blobIdent, error) {
	root, err := rootFromPath(dir)
	if err != nil {
		return namer, err
	}
	namer.root = root
	return namer, nil
}

func populateIndex(namer blobIdent, fname string) (blobIdent, error) {
	idx, err := idxFromPath(fname)
	if err != nil {
		return namer, err
	}
	namer.index = idx
	return namer, nil
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
	idx, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	if idx >= fieldparams.MaxBlobsPerBlock {
		return 0, errors.Wrapf(errIndexOutOfBounds, "index=%d", idx)
	}
	return idx, nil
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
