package filesystem

import (
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

type directoryMigrator interface {
	migrate(fs afero.Fs, dirs []string) error
}

type oneBytePrefixMigrator struct {
	migrated []string
}

func (m *oneBytePrefixMigrator) migrate(fs afero.Fs, dirs []string) error {
	start := time.Now()
	defer func() {
		nMigrated := len(m.migrated)
		if nMigrated > 0 {
			log.WithField("elapsed", time.Since(start).String()).
				WithField("dirsMoved", nMigrated).
				Debug("Migrated blob subdirectories to byte-prefixed format")
		}
	}()
	groups := groupDirsByPrefix(dirs)
	return m.renameByGroup(fs, groups)
}

func (m *oneBytePrefixMigrator) renameByGroup(fs afero.Fs, groups map[string][]string) error {
	for g, sd := range groups {
		// make the enclosing dir if needed
		if err := fs.MkdirAll(g, directoryPermissions); err != nil {
			return err
		}
		for _, dir := range sd {
			dest := filepath.Join(g, dir)
			// todo: check if directory exists and move files one at a time if so?
			// that shouldn't be a problem if we migrate in cache warmup and never write to old path.
			if err := fs.Rename(dir, dest); err != nil {
				return err
			}
			log.WithField("source", dir).WithField("dest", dest).Trace("Migrated legacy blob storage path.")
			m.migrated = append(m.migrated, dir)
		}
	}
	return nil
}

func groupDirsByPrefix(dirs []string) map[string][]string {
	groups := make(map[string][]string)
	for _, dir := range dirs {
		if filterLegacy(dir) {
			key := oneBytePrefix(dir)
			groups[key] = append(groups[key], dir)
		}
	}
	return groups
}
