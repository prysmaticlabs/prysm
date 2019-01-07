package iputils

import (
	"regexp"
	"testing"
)

func TestExternalIPv4(t *testing.T) {
	test, err := ExternalIPv4()

	if err != nil {
		t.Errorf("Test check external ipv4 failed with %g", err)
	}

	valid := regexp.MustCompile(`\.\d{1,3}\.\d{1,3}\b`)

	if !valid.MatchString(test) {
		t.Errorf("Test check check external ipv4, string does not match regexp")
	}
}
