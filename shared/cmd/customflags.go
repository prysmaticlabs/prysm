package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"
)

// DirectoryString -- Custom type which is registered in the flags library
// which cli uses for argument parsing. This allows us to expand Value to
// an absolute path when the argument is parsed.
type DirectoryString struct {
	Value string
}

func (d *DirectoryString) String() string {
	return d.Value
}

// Set directory string value
func (d *DirectoryString) Set(value string) error {
	d.Value = expandPath(value)
	return nil
}

func prefixFor(name string) (prefix string) {
	if len(name) == 1 {
		prefix = "-"
	} else {
		prefix = "--"
	}
	return prefix
}

func prefixedNames(fullName string) (prefixed string) {
	parts := strings.Split(fullName, ",")
	for i, name := range parts {
		name = strings.Trim(name, " ")
		prefixed += prefixFor(name) + name
		if i < len(parts)-1 {
			prefixed += ", "
		}
	}
	return prefixed
}

// DirectoryFlag expands the received string to an absolute path.
// e.g. ~/.ethereum -> /home/username/.ethereum
type DirectoryFlag struct {
	Name  string
	Value DirectoryString
	Usage string
}

func (d DirectoryFlag) String() string {
	fmtString := "%s %v\t%v"
	if len(d.Value.Value) > 0 {
		fmtString = "%s \"%v\"\t%v"
	}
	return fmt.Sprintf(fmtString, prefixedNames(d.Name), d.Value.Value, d.Usage)
}

func eachName(longName string, fn func(string)) {
	parts := strings.Split(longName, ",")
	for _, name := range parts {
		name = strings.Trim(name, " ")
		fn(name)
	}
}

// Apply is called by cli library, grabs variable from environment (if in env)
// and adds variable to flag set for parsing.
func (d DirectoryFlag) Apply(set *flag.FlagSet) {
	eachName(d.Name, func(name string) {
		set.Var(&d.Value, d.Name, d.Usage)
	})
}

// GetName of directory.
func (d DirectoryFlag) GetName() string {
	return d.Name
}

// Set flag value.
func (d *DirectoryFlag) Set(value string) {
	d.Value.Value = value
}

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
