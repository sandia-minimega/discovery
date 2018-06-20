// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"errors"
	"net"

	log "pkg/minilog"
)

type KnownSubnets struct {
	Subnets map[string]*net.IPNet

	// Stores the sorted sizes of all known sizes so that we know how the query
	// IP addresses could be masked. Sorted into descending order so that we
	// have the most specific subnets first.
	Sizes []int
}

func NewKnownSubnets() *KnownSubnets {
	return &KnownSubnets{
		Subnets: make(map[string]*net.IPNet),
		Sizes:   []int{},
	}
}

func (s *KnownSubnets) Add(ipnet *net.IPNet) {
	key := ipnet.String()

	// Already known... don't try to add again
	if _, ok := s.Subnets[key]; ok {
		return
	}

	log.Debug("Adding subnet: %v", key)

	// Check to see if we have overlapping subnets
	for _, other := range s.Subnets {
		// If the added subnet is less specific than one we already have, it's
		// probably an error in the default subnet detection in which case we
		// want to drop it so that it doesn't mess up the results.
		if ipnet.Contains(other.IP) {
			log.Debug("Subnets are nested: %v >> %v", ipnet, other)
			return
		}

		// This is an odd case... we found a subnet more specific that one we
		// already knew about. This is bad and may invalidate our results.
		if other.Contains(ipnet.IP) {
			log.Warn("Subnets are nested: %v << %v", ipnet, other)
		}
	}

	ones, _ := ipnet.Mask.Size()

	s.Subnets[key] = ipnet

	// Insert into sizes to keep it in descending order. Use added to determine
	// whether we have found the largest size so far and thus need to append it
	// to the end.
	var added bool
	for i, v := range s.Sizes {
		if v == ones {
			added = true
			break
		} else if ones > v {
			s.Sizes = append(s.Sizes, 0)
			copy(s.Sizes[i+1:], s.Sizes[i:])
			s.Sizes[i] = ones
			added = true
			break
		}
	}
	if !added {
		s.Sizes = append(s.Sizes, ones)
	}
}

func (s KnownSubnets) Subnet(ip net.IP) (*net.IPNet, error) {
	// Loop over all known subnet sizes
	for _, ones := range s.Sizes {
		mask := net.CIDRMask(ones, 8*net.IPv4len)
		masked := ip.Mask(mask)

		ipnet := &net.IPNet{
			IP:   masked,
			Mask: mask,
		}
		if _, ok := s.Subnets[ipnet.String()]; ok {
			return ipnet, nil
		}
	}

	return nil, errors.New("subnet not known")
}
