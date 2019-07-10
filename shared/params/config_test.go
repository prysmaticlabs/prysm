package params

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestOverrideBeaconConfig(t *testing.T) {
	cfg := BeaconConfig()
	cfg.ShardCount = 5
	OverrideBeaconConfig(cfg)
	if c := BeaconConfig(); c.ShardCount != 5 {
		t.Errorf("Shardcount in BeaconConfig incorrect. Wanted %d, got %d", 5, c.ShardCount)
	}
}

func TestMainnetConfig_ZeroHashIsAliasOfSha256(t *testing.T) {
	h := sha256.New()
	if !bytes.Equal(BeaconConfig().ZeroHash[:], h.Sum(nil)) {
		t.Error("Zero hash is not sha256.New().Sum()")
	}
}
