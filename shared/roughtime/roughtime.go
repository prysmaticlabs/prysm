// Package roughtime is a wrapper for a roughtime clock source
package roughtime

import (
	"encoding/base64"
	"time"

	rt "github.com/cloudflare/roughtime"
	"github.com/sirupsen/logrus"
	"roughtime.googlesource.com/roughtime.git/go/config"
)

// offset is the difference between the system time and the time returned by
// the roughtime server
var offset time.Duration

// Decode or panic
func mustDecodeString(in string) []byte {
	pk, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}
	return pk
}

var log = logrus.WithField("prefix", "roughtime")

func init() {
	t0 := time.Now()

	// A list of reliable roughtime servers with their public keys.
	// From https://github.com/cloudflare/roughtime/blob/master/ecosystem.json
	servers := []config.Server{
		{
			Name:          "Caesium",
			PublicKeyType: "ed25519",
			PublicKey:     mustDecodeString("iBVjxg/1j7y1+kQUTBYdTabxCppesU/07D4PMDJk2WA="),
			Addresses: []config.ServerAddress{
				{
					Protocol: "udp",
					Address:  "caesium.tannerryan.ca:2002",
				},
			},
		},
		{
			Name:          "Chainpoint-Roughtime",
			PublicKeyType: "ed25519",
			PublicKey:     mustDecodeString("bbT+RPS7zKX6w71ssPibzmwWqU9ffRV5oj2OresSmhE="),
			Addresses: []config.ServerAddress{
				{
					Protocol: "udp",
					Address:  "roughtime.chainpoint.org:2002",
				},
			},
		},
		{
			Name:          "Cloudflare-Roughtime",
			PublicKeyType: "ed25519",
			PublicKey:     mustDecodeString("gD63hSj3ScS+wuOeGrubXlq35N1c5Lby/S+T7MNTjxo="),
			Addresses: []config.ServerAddress{
				{
					Protocol: "udp",
					Address:  "roughtime.cloudflare.com:2002",
				},
			},
		},
	}

	results := rt.Do(servers, rt.DefaultQueryAttempts, rt.DefaultQueryTimeout, nil)

	// Compute the average difference between the system's time and the
	// Roughtime responses from the servers, rejecting responses whose radii
	// are larger than 2 seconds.
	var err error
	offset, err = rt.AvgDeltaWithRadiusThresh(results, t0, 2*time.Second)
	if err != nil {
		log.WithError(err).Error("Failed to calculate roughtime offset")
	}
}

// Since returns the duration since t, based on the roughtime response
func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}

// Until returns the duration until t, based on the roughtime response
func Until(t time.Time) time.Duration {
	return t.Sub(Now())
}

// Now returns the current local time given the roughtime offset.
func Now() time.Time {
	return time.Now().Add(offset)
}
