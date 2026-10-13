package features

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/testing/assert"
	"github.com/urfave/cli/v2"
)

func TestWithoutFlags(t *testing.T) {
	a := &cli.BoolFlag{Name: "a"}
	b := &cli.BoolFlag{Name: "b"}
	c := &cli.BoolFlag{Name: "c"}

	t.Run("no excluded flags", func(t *testing.T) {
		assert.DeepEqual(t, []cli.Flag{a, b, c}, WithoutFlags([]cli.Flag{a, b, c}, nil))
	})

	t.Run("some excluded flags", func(t *testing.T) {
		assert.DeepEqual(t, []cli.Flag{b}, WithoutFlags([]cli.Flag{a, b, c}, []cli.Flag{a, c}))
	})

	t.Run("same name but different flag", func(t *testing.T) {
		other := &cli.BoolFlag{Name: "a"}
		assert.DeepEqual(t, []cli.Flag{a}, WithoutFlags([]cli.Flag{a}, []cli.Flag{other}))
	})
}
