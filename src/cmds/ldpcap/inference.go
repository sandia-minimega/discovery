// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"

	log "pkg/minilog"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type OS struct {
	Label string
	Fuzzy bool
}

type Service struct {
	Internet  gopacket.LayerType
	Transport gopacket.LayerType
	Port      uint16
}

type Hostname struct {
	Name string
	Type layers.DNSType
}

type Inference struct {
	ByIP  map[string]*Host
	ByMAC map[string]*Host

	KnownSubnets *KnownSubnets

	AdvertisedServices map[string][]string

	events chan Event

	stats map[string]int
}

func NewInference(events chan Event) *Inference {
	return &Inference{
		ByIP:         make(map[string]*Host),
		ByMAC:        make(map[string]*Host),
		KnownSubnets: NewKnownSubnets(),

		AdvertisedServices: make(map[string][]string),

		events: events,
		stats:  make(map[string]int),
	}
}

func (i *Inference) GetByIP(ip net.IP) *Host {
	key := ip.String()

	if _, ok := i.ByIP[key]; !ok {
		host := NewHost()
		host.External = true
		host.IPs[ip.String()] = ip

		i.ByIP[key] = host
	}

	return i.ByIP[key]
}

func (i *Inference) GetByMAC(mac net.HardwareAddr) *Host {
	key := mac.String()

	if _, ok := i.ByMAC[key]; !ok {
		host := NewHost()
		host.MACs[mac.String()] = mac
		i.ByMAC[key] = host
	}

	return i.ByMAC[key]
}

func (i *Inference) Run() {
	for e := range i.events {
		switch e := e.(type) {
		case *EventService:
			host := i.GetByIP(e.IP)

			host.Services[e.Service] = true
		case *EventAdvertisedService:
			// TODO: This is terrible. Also, store port
			i.AdvertisedServices[e.Hostname] = append(i.AdvertisedServices[e.Hostname], e.Service)

		case *EventHostname:
			host := i.GetByIP(e.IP)

			// TODO: Track DNSType
			host.Hostnames[e.Hostname] = true

			if services, ok := i.AdvertisedServices[e.Hostname.Name]; ok {
				for _, s := range services {
					host.AdvertisedServices[s] = true
				}
				delete(i.AdvertisedServices, e.Hostname.Name)
			}
		case *EventNameserver:
			host := i.GetByIP(e.IP)

			host.Nameservers[e.Nameserver.String()] = true
		case *EventOS:
			host := i.GetByIP(e.IP)

			host.OS[e.OS] += e.Weight
		case *EventDHCP:
			switch e.MsgType {
			case layers.DHCPMsgTypeDiscover:
				// new machine on network
				host := i.GetByMAC(e.HardwareAddr)

				// TODO: Should we record anything about it?
				_ = host
			case layers.DHCPMsgTypeAck, layers.DHCPMsgTypeInform:
				// machine getting info from DHCP server
				host := i.GetByMAC(e.HardwareAddr)

				if e.Hostname != "" {
					hostname := Hostname{
						Name: e.Hostname,
						Type: layers.DNSTypeA,
					}

					host.Hostnames[hostname] = true
				}
				for _, ns := range e.Nameservers {
					host.Nameservers[ns.String()] = true
				}
				for _, r := range e.Routers {
					host.Routers[r.String()] = true

					// Track the provided host as a router
					i.GetByIP(r.IP).Router = true
				}

				if e.Subnet.IP != nil && e.Subnet.Mask != nil {
					i.KnownSubnets.Add(&e.Subnet)
				}

				if !e.ClientIP.IsUnspecified() {
					// TODO: We don't necessarily want to remember that this host
					// had this IP indefinitely...
					host.IPs[e.ClientIP.String()] = e.ClientIP

					// Track that this host is assigned to this IP
					i.ByIP[e.ClientIP.String()] = host
				}
			}
		case *EventNeighbor:
			host := i.GetByMAC(e.HardwareAddr)

			// TODO: We don't necessarily want to remember that this host
			// had this IP indefinitely...
			if _, ok := host.IPs[e.IP.String()]; !ok {
				host.IPs[e.IP.String()] = e.IP
			}

			// Track that this host is assigned to this IP
			i.ByIP[e.IP.String()] = host
		case *EventRouter:
			if e.HardwareAddr != nil {
				host := i.GetByMAC(e.HardwareAddr)

				// TODO: We don't necessarily want to remember that this host
				// had this IP indefinitely...
				if _, ok := host.IPs[e.IP.String()]; !ok {
					host.IPs[e.IP.String()] = e.IP
				}

				host.Router = true
			}

			for _, ipp := range e.IPPrefixes {
				i.KnownSubnets.Add(&ipp)
			}
		default:
			log.Info("Unhandled event: %#v", e)
		}
	}
}

func (i *Inference) WriteHosts(out io.Writer) {
	count := 0

	for _, host := range i.ByMAC {
		fmt.Fprintf(out, "[%v]\n", count)

		host.Write(out)

		count += 1
	}

	for _, host := range i.ByIP {
		// Internal hosts have already been printed in the ByMAC loop
		if host.External {
			fmt.Fprintf(out, "[%v]\n", count)

			host.Write(out)

			count += 1
		}
	}
}

func (i *Inference) WriteHostsJSON(out io.Writer) {
	hosts := []*HostOut{}

	for _, host := range i.ByMAC {
		hosts = append(hosts, host.Out())
	}

	for _, host := range i.ByIP {
		// Internal hosts have already been printed in the ByMAC loop
		if host.External {
			hosts = append(hosts, host.Out())
		}
	}

	json.NewEncoder(out).Encode(hosts)
}
