package forks

import (
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"sort"
)

// ForkScheduleEntry is a Version+Epoch tuple for sorted storage in an OrderedSchedule
type ForkScheduleEntry struct {
	Version [fieldparams.VersionLength]byte
	Epoch   types.Epoch
}

// OrderedSchedule provides a type that can be used to sort the fork schedule and find the Version
// the chain should be at for a given epoch (via VersionForEpoch).
type OrderedSchedule []ForkScheduleEntry

// Len implements the Len method of sort.Interface
func (o OrderedSchedule) Len() int { return len(o) }

// Swap implements the Swap method of sort.Interface
func (o OrderedSchedule) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

// Less implements the Less method of sort.Interface
func (o OrderedSchedule) Less(i, j int) bool { return o[i].Epoch < o[j].Epoch }

// VersionForEpoch finds the Version with the highest epoch <= the given epoch
func (o OrderedSchedule) VersionForEpoch(epoch types.Epoch) ([fieldparams.VersionLength]byte, error) {
	for i := len(o) - 1; i >= 0; i-- {
		if o[i].Epoch <= epoch {
			return o[i].Version, nil
		}
	}
	return [fieldparams.VersionLength]byte{}, errors.Wrapf(ErrVersionNotFound, "no epoch in list <= %d", epoch)
}

// Converts the ForkVersionSchedule map into a list of Version+Epoch values, ordered by Epoch from lowest to highest.
// See docs for OrderedSchedule for more detail on what you can do with this type.
func NewOrderedSchedule(b *params.BeaconChainConfig) OrderedSchedule {
	ofs := make(OrderedSchedule, 0)
	for version, epoch := range b.ForkVersionSchedule {
		fse := ForkScheduleEntry{
			Version: version,
			Epoch:   epoch,
		}
		ofs = append(ofs, fse)
	}
	sort.Sort(ofs)
	return ofs
}
