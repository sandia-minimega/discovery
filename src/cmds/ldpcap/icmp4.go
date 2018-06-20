// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"github.com/google/gopacket/layers"
)

func (s *State) HandleICMPv4() {
	switch s.icmp4.TypeCode {
	case layers.ICMPv4TypeEchoRequest:
		//fmt.Printf("ICMPv4 ping -- %v -> %v %v\n", s.ip4.SrcIP, s.ip4.DstIP, s.ip4.TTL)
		// TODO: Ping/pong => connectivity testing, useful?
	case layers.ICMPv4TypeEchoReply:
		//fmt.Printf("ICMPv4 pong -- %v -> %v %v\n", s.ip4.SrcIP, s.ip4.DstIP, s.ip4.TTL)
		// TODO: Ping/pong => connectivity testing, useful?
	case layers.ICMPv4TypeTimeExceeded:
		//fmt.Printf("ICMPv4 time exceeded -- %v -> %v %v\n", s.ip4.SrcIP, s.ip4.DstIP, s.ip4.TTL)
		switch uint8(s.icmp4.TypeCode) {
		case layers.ICMPv4CodeTTLExceeded:
			// TODO: maybe interesting
		}
	case layers.ICMPv4TypeDestinationUnreachable:
		//fmt.Printf("ICMPv4 destination unreachable -- %v -> %v %v\n", s.ip4.SrcIP, s.ip4.DstIP, s.icmp4.TypeCode)
		switch uint8(s.icmp4.TypeCode) {
		case layers.ICMPv4CodeNet:
			// TODO: maybe interesting
		case layers.ICMPv4CodeHost:
			// TODO: maybe interesting
		case layers.ICMPv4CodePort:
			// TODO: maybe interesting
		case layers.ICMPv4CodeFragmentationNeeded:
			// TODO: maybe interesting
		case layers.ICMPv4CodeNetUnknown:
			// TODO: maybe interesting
		case layers.ICMPv4CodeHostUnknown:
			// TODO: maybe interesting
		}
	}
}
