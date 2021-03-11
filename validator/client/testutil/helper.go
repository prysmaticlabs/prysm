package testutil

import "github.com/prysmaticlabs/prysm/shared/bytesutil"

// ActiveKey represents a public key whose status is ACTIVE.
var ActiveKey = bytesutil.ToBytes48([]byte("active"))
