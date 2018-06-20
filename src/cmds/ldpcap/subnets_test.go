// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"net"
	"testing"
)

var subnets = []struct {
	CIDR      string
	IP        string
	Contained bool
}{
	{"192.168.0.0/16", "192.168.10.10", true},
	{"192.168.10.0/24", "192.168.10.10", true},
	{"192.168.10.0/24", "192.168.20.10", false},
	{"192.168.10.32/27", "192.168.10.34", true},
}

func TestKnownSubnets(t *testing.T) {
	for _, subnet := range subnets {

		_, ipnet, err := net.ParseCIDR(subnet.CIDR)
		if err != nil {
			t.Errorf("unable to parse CIDR: %v", err)
			continue
		}

		ip := net.ParseIP(subnet.IP)
		if ipnet.Contains(ip) != subnet.Contained {
			t.Errorf("%v <-> %v != %v", ipnet, ip, subnet.Contained)
			continue
		}

		s := NewKnownSubnets()
		s.Add(ipnet)

		res, err := s.Subnet(ip)
		if subnet.Contained {
			if err != nil {
				t.Errorf("error getting subnet: %v", err)
				continue
			}

			if res.String() != ipnet.String() {
				t.Errorf("%v != %v", res, ipnet)
			}
		} else if err == nil {
			t.Errorf("should have failed to find subnet: %v", res)
		}
	}
}
