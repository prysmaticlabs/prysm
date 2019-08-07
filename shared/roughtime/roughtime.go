// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package roughtime is a wrapper for a roughtime clock source
package roughtime

import (
	"fmt"
	"time"
	"encoding/base64"

	"github.com/cloudflare/roughtime"
	"roughtime.googlesource.com/roughtime.git/go/config"
)

// offset is the difference between the system time and the time returned by
// the roughtime server
var offset time.Duration

func init() {
	pk, err := base64.StdEncoding.DecodeString("gD63hSj3ScS+wuOeGrubXlq35N1c5Lby/S+T7MNTjxo=")
	if err != nil {
		panic(err)
	}

	server := &config.Server{
		Name:          "",
		PublicKeyType: "ed25519",
		PublicKey:     pk,
		Addresses: []config.ServerAddress{
			config.ServerAddress{
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
	fmt.Println("init:", time.Now(), rtTime, offset)
}

// Since returns the duration since t, based on the roughtime response
func Since(t time.Time) time.Duration {
	fmt.Println("since:", time.Now(), time.Now().Add(offset), time.Now().Add(offset).Sub(t))
	return time.Now().Add(offset).Sub(t)
}

// Until returns the duration until t, based on the roughtime response
func Until(t time.Time) time.Duration {
	return t.Sub(time.Now().Add(offset))
}
