package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var errUsage = errors.New("incorrect usage - missing blob path")

const newLen = 4 // eg '0xff'

func blobPath() (string, error) {
	if len(os.Args) < 2 {
		return "", errUsage
	}
	return os.Args[1], nil
}

func usage(err error) {
	fmt.Printf("%s\n", err.Error())
	fmt.Println("downgrade-blob-storage: Move blob directories back to old format, without the single byte container directories at the top-level of the directory tree. usage:\n" + os.Args[0] + " <path to blobs dir>")
}

func main() {
	bp, err := blobPath()
	if err != nil {
		if errors.Is(err, errUsage) {
			usage(err)
		}
		os.Exit(1)
	}
	if err := downgrade(bp); err != nil {
		fmt.Printf("fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}

func downgrade(base string) error {
	top, err := os.Open(base) // #nosec G304
	if err != nil {
		return err
	}
	// iterate over top-level blob dir, ie 'blobs' inside prysm's datadir
	topdirs, err := top.Readdirnames(0)
	if err != nil {
		return err
	}
	if err := top.Close(); err != nil {
		return err
	}
	nMoved := 0
	for _, td := range topdirs {
		// Ignore anything in the old layout.
		if !filterNew(td) {
			continue
		}
		dir, err := os.Open(filepath.Join(base, td)) // #nosec G304
		if err != nil {
			return err
		}
		// List the subdirectoress of the short dir containers, eg if td == '0xff'
		// we want to move all the subdirectories in that dir.
		subs, err := dir.Readdirnames(0)
		if err != nil {
			return err
		}
		if err := dir.Close(); err != nil {
			return err
		}
		for _, sd := range subs {
			// this is the inner layer of directory nesting,
			// eg if 'td' == '0xff', 'sd' might be something like:
			// '0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb'
			src := filepath.Join(base, td, sd)
			target := filepath.Join(base, sd)
			fmt.Printf("moving %s -> %s\n", src, target)
			if err := os.Rename(src, target); err != nil {
				return err
			}
			nMoved += 1
		}
	}
	fmt.Printf("moved %d directories\n", nMoved)
	return nil
}

func filterRoot(s string) bool {
	return strings.HasPrefix(s, "0x")
}

func filterNew(s string) bool {
	return filterRoot(s) && len(s) == newLen
}
