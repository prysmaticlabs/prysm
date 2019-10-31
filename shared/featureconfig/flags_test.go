package featureconfig

import (
	"reflect"
	"strings"
	"testing"
)

func TestDeprecatedFlags(t *testing.T) {
	for _, f := range deprecatedFlags {
		fv := reflect.ValueOf(f)
		if !fv.FieldByName("Hidden").Bool() {
			t.Errorf("%s must be hidden when deprecated.", f.GetName())
		}
		if !strings.Contains(fv.FieldByName("Usage").String(), "DEPRECATED. DO NOT USE.") {
			t.Errorf("Usage for %s must contain \"DEPRECATED. DO NOT USE.\"", f.GetName())
		}
	}
}
