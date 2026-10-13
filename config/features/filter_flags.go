package features

import (
	"reflect"
	"slices"

	"github.com/urfave/cli/v2"
)

// ActiveFlags returns all of the flags that are not Hidden.
func ActiveFlags(flags []cli.Flag) []cli.Flag {
	visible := make([]cli.Flag, 0, len(flags))
	for _, flag := range flags {
		field := flagValue(flag).FieldByName("Hidden")
		if !field.IsValid() || !field.Bool() {
			visible = append(visible, flag)
		}
	}
	return visible
}

// WithoutFlags returns all of the flags that are not in excluded.
func WithoutFlags(flags, excluded []cli.Flag) []cli.Flag {
	kept := make([]cli.Flag, 0, len(flags))
	for _, flag := range flags {
		if !slices.Contains(excluded, flag) {
			kept = append(kept, flag)
		}
	}
	return kept
}

func flagValue(f cli.Flag) reflect.Value {
	fv := reflect.ValueOf(f)
	for fv.Kind() == reflect.Pointer {
		fv = reflect.Indirect(fv)
	}
	return fv
}
