/**
 * File        : util.go
 * Description : Functions involving hashes.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 *
 * This module provides some commonly-used functions involving hashes.
 */

package bls

import (
	"crypto/rand"
	"crypto/sha256"
)

func randomHash() ([sha256.Size]byte, error) {
	var hash [sha256.Size]byte
	_, err := rand.Read(hash[:])
	return hash, err
}

func randomHashes(n int) ([][sha256.Size]byte, error) {
	hashes := make([][sha256.Size]byte, n)
	var err error
	for i := range hashes {
		hashes[i], err = randomHash()
		if err != nil {
			return nil, err
		}
	}
	return hashes, nil
}

func sortHashes(hashes [][sha256.Size]byte) {
	n := len(hashes)
	quicksort(hashes, 0, n-1)
}

func quicksort(hashes [][sha256.Size]byte, l int, r int) {
	if l < r {
		pivot := hashes[(l+r)/2]
		var i, j int
		for i, j = l, r; i <= j; i, j = i+1, j-1 {
			for ; compare(hashes[i], pivot) == -1; i++ {
			}
			for ; compare(hashes[j], pivot) == 1; j-- {
			}
			if i < j {
				hashes[i], hashes[j] = hashes[j], hashes[i]
			} else if i != j {
				break
			}
		}
		quicksort(hashes, l, j)
		quicksort(hashes, i, r)
	}
}

func compare(a, b [sha256.Size]byte) int {
	for i := 0; i < sha256.Size; i++ {
		if a[i] > b[i] {
			return 1
		}
		if a[i] < b[i] {
			return -1
		}
	}
	return 0
}

func uniqueHashes(hashes [][sha256.Size]byte) bool {
	n := len(hashes)
	c := make([][sha256.Size]byte, n)
	copy(c, hashes)
	sortHashes(c)
	for i := 0; i < n-1; i++ {
		if c[i] == c[i+1] {
			return false
		}
	}
	return true
}
