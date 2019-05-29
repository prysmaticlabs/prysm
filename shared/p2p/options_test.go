package p2p

import (
	"testing"
)

func TestBuildOptions(t *testing.T) {
	opts, _ := buildOptions(&ServerConfig{})

	_ = opts
}
