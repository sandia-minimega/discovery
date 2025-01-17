// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"strings"

	"github.com/google/gopacket/layers"
)

func (s *State) HandleDNS() {
	// QR => Response
	if s.dns.QR {
		s.HandleDNSResponse()
	} else {
		s.HandleDNSQuery()
	}
}

func (s *State) HandleDNSResponse() {
	for _, answer := range s.dns.Answers {
		switch answer.Type {
		// Emit Hostname events for A, AAAA, NS, and MX records
		case layers.DNSTypeA, layers.DNSTypeAAAA, layers.DNSTypeMX, layers.DNSTypeNS:
			s.events <- &EventHostname{
				IP: answer.IP,
				Hostname: Hostname{
					// Hostnames are case insensitive (RFC 4343)
					Name: strings.ToLower(string(answer.Name)),
					Type: answer.Type,
				},
			}
		// Emit AdvertisedServices events for SRV records
		case layers.DNSTypeSRV:
			s.events <- &EventAdvertisedService{
				Service:  string(answer.Name),
				Hostname: string(answer.SRV.Name),
				Port:     answer.SRV.Port,
			}
		}
	}
}

func (s *State) HandleDNSQuery() {
	ns := s.DstIP()

	// Don't track multicast IPs used in mDNS
	if !ns.IsLinkLocalMulticast() {
		// Emit Nameserver event to track what nameservers host uses
		s.events <- &EventNameserver{
			IP:         s.SrcIP(),
			Nameserver: ns,
		}
	}
}
