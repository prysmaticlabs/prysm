// Package iputils contains useful functions for ip address formatting.
package iputils

import (
	"net"
)

// ExternalIPv4 returns the first IPv4 available.
func ExternalIPv4() (string, error) {
	ips, err := retrieveIPAddrs()
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "127.0.0.1", nil
	}
	for _, ip := range ips {
		ip = ip.To4()
		if ip == nil {
			continue // not an ipv4 address
		}
		return ip.String(), nil
	}
	return "127.0.0.1", nil
}

// ExternalIP returns the first IPv4/IPv6 available.
func ExternalIP() (string, error) {
	ips, err := retrieveIPAddrs()
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "127.0.0.1", nil
	}
	return ips[0].String(), nil
}

// retrieveIP returns all the valid IPs available.
func retrieveIPAddrs() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	ipAddrs := []net.IP{}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ipAddrs = append(ipAddrs, ip)
		}
	}
	return ipAddrs, nil
}
