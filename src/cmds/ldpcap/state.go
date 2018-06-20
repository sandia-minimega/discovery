// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type State struct {
	*gopacket.DecodingLayerParser
	captureInfo gopacket.CaptureInfo

	eth   layers.Ethernet
	dot1q layers.Dot1Q
	ip4   layers.IPv4
	ip6   layers.IPv6
	icmp4 layers.ICMPv4
	tcp   layers.TCP
	udp   layers.UDP
	dns   layers.DNS
	arp   layers.ARP
	dhcp  layers.DHCPv4

	link      gopacket.LayerType
	internet  gopacket.LayerType
	transport gopacket.LayerType

	events chan Event
}

func (s *State) IP() gopacket.Layer {
	switch s.internet {
	case layers.LayerTypeIPv4:
		return &s.ip4
	case layers.LayerTypeIPv6:
		return &s.ip6
	}

	return nil
}

func (s *State) TCP() *layers.TCP {
	return &s.tcp
}
