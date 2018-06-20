// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"net"

	"pkg/minigraph"
)

func UpdateEndpoint(v *minigraph.Endpoint) error {
	if *f_dryrun {
		return nil
	}

	_, err := dc.UpdateEndpoints(v)
	return err
}

// GetIPs parses the IP field of all edges for a node.
func GetIPs(v *minigraph.Endpoint) []net.IP {
	ips := []net.IP{}

	for _, edg := range v.Edges {
		if i, ok := edg.D["ip"]; ok {
			v, _, err := net.ParseCIDR(i)
			if err == nil {
				ips = append(ips, v)
			} else if ip := net.ParseIP(i); ip != nil {
				ips = append(ips, ip)
			}
		}
	}

	return ips
}
