package featureconfig

import (
	"reflect"
	"strings"
	"testing"
)

func TestDeprecatedFlags(t *testing.T) {
	for _, f := range deprecatedFlags {
		fv := reflect.ValueOf(f)
		field := reflect.Indirect(fv).FieldByName("Hidden")
		if !field.IsValid() || !field.Bool() {
			t.Errorf("%s must be hidden when deprecated.", f.Names()[0])
		}
		if !strings.Contains(reflect.Indirect(fv).FieldByName("Usage").String(), "DEPRECATED. DO NOT USE.") {
			t.Errorf("Usage for %s must contain \"DEPRECATED. DO NOT USE.\"", f.Names()[0])
		}
	}
}
