// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"net"
	"strings"

	"github.com/google/gopacket/layers"
)

func (s *State) HandleDHCP() {
	e := &EventDHCP{
		HardwareAddr: s.dhcp.ClientHWAddr,
		ClientIP:     s.dhcp.ClientIP,
	}

	for _, option := range s.dhcp.Options {
		switch option.Type {
		case layers.DHCPOptMessageType:
			if len(option.Data) != 1 {
				continue
			}

			e.MsgType = layers.DHCPMsgType(option.Data[0])

			switch e.MsgType {
			case layers.DHCPMsgTypeOffer, layers.DHCPMsgTypeAck:
				if e.ClientIP.IsUnspecified() {
					e.ClientIP = s.dhcp.YourClientIP
				}

				// Double check that the ClientIP is now specified. Seen many
				// cases where neither ClientIP nor YourClientIP is set... not
				// sure what that means.
				if !e.ClientIP.IsUnspecified() {
					// Set default subnet based on default mask. This is the
					// same thing that Linux dhcp clients do when there is no
					// Subnet Option.
					mask := e.ClientIP.DefaultMask()

					e.Subnet = net.IPNet{
						IP:   e.ClientIP.Mask(mask),
						Mask: mask,
					}
				}
			}
		case layers.DHCPOptSubnetMask:
			mask := net.IPMask(option.Data)

			e.Subnet = net.IPNet{
				IP:   e.ClientIP.Mask(mask),
				Mask: mask,
			}
		case layers.DHCPOptTimeOffset:
		case layers.DHCPOptRouter:
			data := option.Data
			for len(data) > 0 {
				e.Routers = append(e.Routers, net.IPNet{
					IP: net.IP(data[:4]),
				})
				data = data[4:]
			}
		case layers.DHCPOptTimeServer:
		case layers.DHCPOptNameServer:
		case layers.DHCPOptDNS:
			data := option.Data
			for len(data) > 0 {
				e.Nameservers = append(e.Nameservers, net.IP(data[:4]))
				data = data[4:]
			}
		case layers.DHCPOptLogServer:
		case layers.DHCPOptCookieServer:
		case layers.DHCPOptLPRServer:
		case layers.DHCPOptImpressServer:
		case layers.DHCPOptResLocServer:
		case layers.DHCPOptHostname:
			// Hostnames are case insensitive (RFC 4343)
			e.Hostname = strings.ToLower(string(option.Data))
		case layers.DHCPOptBootfileSize:
		case layers.DHCPOptMeritDumpFile:
		case layers.DHCPOptDomainName:
			e.Domain = string(option.Data)
		case layers.DHCPOptSwapServer:
		case layers.DHCPOptRootPath:
		case layers.DHCPOptExtensionsPath:
		case layers.DHCPOptRequestIP:
			e.RequestedIPAddr = net.IP(option.Data)
		}
	}

	// Patch up the routers with the subnet mask
	for i := range e.Routers {
		e.Routers[i].Mask = e.Subnet.Mask
	}

	// Patch up the hostname to make a FQDN
	if e.Domain != "" && e.Hostname != "" && !strings.Contains(e.Hostname, ".") {
		e.Hostname = e.Hostname + "." + e.Domain
	}

	s.events <- e
}
