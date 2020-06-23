package iputils_test

import (
	"net"
	"regexp"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/iputils"
)

func TestExternalIPv4(t *testing.T) {
	// Regular expression format for IPv4
	IPv4Format := `\.\d{1,3}\.\d{1,3}\b`
	test, err := iputils.ExternalIPv4()

	if err != nil {
		t.Errorf("Test check external ipv4 failed with %v", err)
	}

	valid := regexp.MustCompile(IPv4Format)

	if !valid.MatchString(test) {
		t.Errorf("Wanted: %v, got: %v", IPv4Format, test)
	}
}

func TestRetrieveIP(t *testing.T) {
	ip, err := iputils.ExternalIP()
	if err != nil {
		t.Fatal(err)
	}
	retIP := net.ParseIP(ip)
	if retIP.To4() == nil && retIP.To16() == nil {
		t.Errorf("An invalid IP was retrieved: %s", ip)
	}
}
