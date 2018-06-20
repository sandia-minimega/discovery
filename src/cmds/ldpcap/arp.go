// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"net"

	"github.com/google/gopacket/layers"
)

func IsUnspecifiedMAC(addr net.HardwareAddr) bool {
	for _, v := range addr {
		if v != 0 {
			return false
		}
	}

	return true
}

func (s *State) HandleARP() {
	// For now, only care about IPv4
	if s.arp.Protocol != layers.EthernetTypeIPv4 {
		return
	}

	srcAddr := net.HardwareAddr(s.arp.SourceHwAddress)
	srcIP := net.IP(s.arp.SourceProtAddress)
	dstAddr := net.HardwareAddr(s.arp.DstHwAddress)
	dstIP := net.IP(s.arp.DstProtAddress)

	if s.arp.Operation == layers.ARPReply {
		if !dstIP.IsUnspecified() && !IsUnspecifiedMAC(dstAddr) {
			s.events <- &EventNeighbor{
				HardwareAddr: dstAddr,
				IP:           dstIP,
			}
		}
	}

	if !srcIP.IsUnspecified() && !IsUnspecifiedMAC(srcAddr) {
		s.events <- &EventNeighbor{
			HardwareAddr: srcAddr,
			IP:           srcIP,
		}
	}
}
