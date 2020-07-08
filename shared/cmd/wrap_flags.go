package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

// WrapFlags so that they can be loaded from alternative sources.
func WrapFlags(flags []cli.Flag) []cli.Flag {
	wrapped := make([]cli.Flag, 0, len(flags))
	for _, f := range flags {
		switch f.(type) {
		case *cli.BoolFlag:
			f = altsrc.NewBoolFlag(f.(*cli.BoolFlag))
		case *cli.DurationFlag:
			f = altsrc.NewDurationFlag(f.(*cli.DurationFlag))
		case *cli.GenericFlag:
			f = altsrc.NewGenericFlag(f.(*cli.GenericFlag))
		case *cli.Float64Flag:
			f = altsrc.NewFloat64Flag(f.(*cli.Float64Flag))
		case *cli.IntFlag:
			f = altsrc.NewIntFlag(f.(*cli.IntFlag))
		case *cli.StringFlag:
			f = altsrc.NewStringFlag(f.(*cli.StringFlag))
		case *cli.StringSliceFlag:
			f = altsrc.NewStringSliceFlag(f.(*cli.StringSliceFlag))
		case *cli.Uint64Flag:
			f = altsrc.NewUint64Flag(f.(*cli.Uint64Flag))
		case *cli.UintFlag:
			f = altsrc.NewUintFlag(f.(*cli.UintFlag))
		case *cli.Int64Flag:
			// Int64Flag does not work. See https://github.com/prysmaticlabs/prysm/issues/6478
			panic(fmt.Sprintf("unsupported flag type type %T", f))
		default:
			panic(fmt.Sprintf("cannot convert type %T", f))
		}
		wrapped = append(wrapped, f)
	}
	return wrapped
}
