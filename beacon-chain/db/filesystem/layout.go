package filesystem

import (
	"github.com/spf13/afero"
)

var defaultLayout = &periodicEpochLayout{}

var allLayouts = []fsLayout{
	defaultLayout,
	&hexPrefixLayout{},
	&legacyLayout{},
}

type fsLayout interface {
	IterateNamers(fs afero.Fs) chan blobNamer
	Detect(baseEntries []string) bool
	Initialize(fs afero.Fs) error
}

func detectLayout(fs afero.Fs) (fsLayout, error) {
	fh, err := fs.Open(".")
	if err != nil {
		return nil, err
	}
	entries, err := fh.Readdirnames(3)
	if err != nil {
		return nil, err
	}
	for _, layout := range allLayouts {
		if layout.Detect(entries) {
			return layout, nil
		}
	}
	// Since we didn't detect the default layout's base directory we can try to initialize it.
	return defaultLayout, defaultLayout.Initialize(fs)
}

type legacyLayout struct{}

func (l *legacyLayout) IterateNamers(fs afero.Fs) chan blobNamer {
	return make(chan blobNamer)
}

func (l *legacyLayout) Detect(entries []string) bool {
	for _, entry := range entries {
		if filterLegacy(entry) {
			return true
		}
	}
	return false
}

func (l *legacyLayout) Initialize(fs afero.Fs) error {
	return nil
}

type hexPrefixLayout struct{}

func (l *hexPrefixLayout) Detect(entries []string) bool {
	for _, entry := range entries {
		if entry == hexPrefixBaseDir {
			return true
		}
	}
	return false
}

func (l *hexPrefixLayout) Initialize(fs afero.Fs) error {
	return fs.MkdirAll(hexPrefixBaseDir, directoryPermissions)
}

func (l *hexPrefixLayout) IterateNamers(fs afero.Fs) chan blobNamer {
	return make(chan blobNamer)
}

type periodicEpochLayout struct{}

func (l *periodicEpochLayout) Detect(entries []string) bool {
	for _, entry := range entries {
		if entry == periodicEpochBaseDir {
			return true
		}
	}
	return false
}

func (l *periodicEpochLayout) Initialize(fs afero.Fs) error {
	return fs.MkdirAll(periodicEpochBaseDir, directoryPermissions)
}

func (l *periodicEpochLayout) IterateNamers(fs afero.Fs) chan blobNamer {
	return make(chan blobNamer)
}
