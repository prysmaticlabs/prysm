package blst

// Note: These functions are for tests to access private globals, such as pubkeyMap.

// KeyMap returns the pubkey cache.
func KeyMap() map[[48]byte]*PublicKey {
	return pubkeyMap
}
