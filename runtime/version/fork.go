package version

import (
	"sort"

	"github.com/pkg/errors"
)

const (
	Phase0 = iota
	Altair
	Bellatrix
	Capella
	Deneb
)

var versionToString = map[int]string{
	Phase0:    "phase0",
	Altair:    "altair",
	Bellatrix: "bellatrix",
	Capella:   "capella",
	Deneb:     "deneb",
}

// stringToVersion and allVersions are populated in init()
var stringToVersion = map[string]int{}
var allVersions = []int{}

// ErrUnrecognizedVersionName means a string does not match the list of canonical version names.
var ErrUnrecognizedVersionName = errors.New("version name doesn't map to a known value in the enum")

// FromString translates a canonical version name to the version number.
func FromString(name string) (int, error) {
	v, ok := stringToVersion[name]
	if !ok {
		return 0, errors.Wrap(ErrUnrecognizedVersionName, name)
	}
	return v, nil
}

// String returns the canonical string form of a version.
// Unrecognized versions won't generate an error and are represented by the string "unknown version".
func String(version int) string {
	name, ok := versionToString[version]
	if !ok {
		return "unknown version"
	}
	return name
}

// All returns a list of all known fork versions.
func All() []int {
	return allVersions
}

func CreateVersions() ([]int, map[string]int) {
	versions := make([]int, len(versionToString))
	versionMap := map[string]int{}

	// range map is random, so we need to sort the keys
	var keys []int
	for k := range versionToString {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	i := 0
	for _, k := range keys {
		versions[i] = k
		versionMap[versionToString[k]] = k
		i++
	}

	return versions, versionMap
}

func init() {
	allVersions, stringToVersion = CreateVersions()
}
