package filesystem

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

const (
	sszExt               = "ssz"
	partExt              = "part"
	rootPrefixLen        = 4
	periodicEpochBaseDir = "by-epoch"
	hexPrefixBaseDir     = "by-hex-prefix"
)

var (
	errInvalidRootString      = errors.New("Could not parse hex string as a [32]byte")
	errInvalidDirectoryLayout = errors.New("Could not parse blob directory path")
)

type blobNamer struct {
	version string
	root    [32]byte
	period  primitives.Epoch
	slot    primitives.Slot
	epoch   primitives.Epoch
	index   uint64
}

func namerForSidecar(sc blocks.VerifiedROBlob) blobNamer {
	slot := sc.Slot()
	epoch := slots.ToEpoch(slot)
	period := epoch / params.BeaconConfig().MinEpochsForBlobsSidecarsRequest

	return blobNamer{version: periodicEpochBaseDir,
		root:   sc.BlockRoot(),
		slot:   slot,
		epoch:  epoch,
		period: period,
		index:  sc.Index,
	}
}

func (p blobNamer) groupDir() string {
	return oneBytePrefix(rootString(p.root))
}

func (p blobNamer) dir() string {
	rs := rootString(p.root)
	parentDir := oneBytePrefix(rs)
	return filepath.Join(parentDir, rs)
}

func (p blobNamer) partPath(entropy string) string {
	return path.Join(p.dir(), fmt.Sprintf("%s-%d.%s", entropy, p.index, partExt))
}

func (p blobNamer) path() string {
	return path.Join(p.dir(), fmt.Sprintf("%d.%s", p.index, sszExt))
}

func rootString(root [32]byte) string {
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

func oneBytePrefix(p string) string {
	// returns eg 0x00 from 0x0002fb4db510b8618b04dc82d023793739c26346a8b02eb73482e24b0fec0555
	return p[0:rootPrefixLen]
}

func namerFromDir(dir string) (blobNamer, error) {
	// ex: by-epoch/66/273848/0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb
	//     ^0       ^1 ^2     ^3
	parts := filepath.SplitList(dir)
	version := parts[0]
	if len(parts) < 4 {
		return blobNamer{}, errInvalidDirectoryLayout
	}
	if version != periodicEpochBaseDir {
		return blobNamer{}, errInvalidDirectoryLayout
	}
	period, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return blobNamer{}, errors.Wrapf(errInvalidDirectoryLayout,
			"failed to decode period as uint, err=%s, dir=%s", err.Error(), dir)
	}
	epoch, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return blobNamer{}, errors.Wrapf(errInvalidDirectoryLayout,
			"failed to decode epoch as uint, err=%s, dir=%s", err.Error(), dir)
	}
	root, err := stringToRoot(parts[3])
	if err != nil {
		return blobNamer{}, errors.Wrapf(errInvalidDirectoryLayout,
			"failed to decode root, err=%s, dir=%s", err.Error(), dir)
	}
	return blobNamer{
		version: version,
		root:    root,
		period:  primitives.Epoch(period),
		epoch:   primitives.Epoch(epoch),
		index:   0,
	}, nil
}
