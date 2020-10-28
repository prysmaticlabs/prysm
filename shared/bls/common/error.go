package common

import "errors"

// ErrZeroKey describes an error due to a zero secret key.
var ErrZeroKey = errors.New("received secret key is zero")

// ErrInfinitePubKey describes an error due to an infinite public key.
var ErrInfinitePubKey = errors.New("received an infinite public key")

// ErrInfiniteSignature describes an error due to an infinite signature.
var ErrInfiniteSignature = errors.New("received an infinite signature")
