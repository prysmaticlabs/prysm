// Package roughtime is a wrapper for a roughtime clock source
package roughtime

import (
	"encoding/base64"
	"time"

	"github.com/cloudflare/roughtime"
	"roughtime.googlesource.com/roughtime.git/go/config"
)

// offset is the difference between the system time and the time returned by
// the roughtime server
var offset time.Duration

func init() {
	// This is the public key used to check roughtime.cloudflare.com:2002 responses
	// See https://github.com/cloudflare/roughtime/blob/master/README.md
	pk, err := base64.StdEncoding.DecodeString("gD63hSj3ScS+wuOeGrubXlq35N1c5Lby/S+T7MNTjxo=")
	if err != nil {
		panic(err)
	}

	server := &config.Server{
		Name:          "",
		PublicKeyType: "ed25519",
		PublicKey:     pk,
		Addresses: []config.ServerAddress{
			{
				Protocol: "udp",
				Address:  "roughtime.cloudflare.com:2002",
			},
		},
	}

	rt, err := roughtime.Get(server, roughtime.DefaultQueryAttempts, roughtime.DefaultQueryTimeout, nil)
	if err != nil {
		panic(err)
	}
	rtTime, _ := rt.Now()
	offset = rtTime.Sub(time.Now())
}

// Since returns the duration since t, based on the roughtime response
func Since(t time.Time) time.Duration {
	return time.Now().Add(offset).Sub(t)
}

// Until returns the duration until t, based on the roughtime response
func Until(t time.Time) time.Duration {
	return t.Sub(time.Now().Add(offset))
}
