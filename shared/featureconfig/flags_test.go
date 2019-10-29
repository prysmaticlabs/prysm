package featureconfig

import (
	"reflect"
	"testing"
)

func TestDeprecatedFlags(t *testing.T) {
	for _, f := range deprecatedFlags {
		fv := reflect.ValueOf(f)
		if !fv.FieldByName("Hidden").Bool() {
			t.Errorf("%s must be hidden when deprecated.", f.GetName())
		}
	}
}
