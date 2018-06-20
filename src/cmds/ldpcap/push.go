// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

func pushHosts() {
	f, err := os.Open(*f_hosts)
	if err != nil {
		log.Fatal("unable to read hosts: %v", err)
	}

	hosts := []*HostOut{}
	if err := json.NewDecoder(f).Decode(&hosts); err != nil {
		log.Fatal("unable to decode hosts: %v", err)
	}

	dc := discovery.New(*f_push)

	for _, h := range hosts {
		if h.External || h.Router {
			continue
		}

		e := &minigraph.Endpoint{}
		es, err := dc.InsertEndpoints(e)
		if err != nil {
			log.Fatalln(err)
		}
		e = es[0]

		if len(h.MACs) > 1 {
			log.Info("found machine with more than one MAC: %v", h.MACs)
		}

		ips := []net.IP{}
		for _, v := range h.IPs {
			ip := net.ParseIP(v)
			if ip.To4() == nil || ip.IsLinkLocalUnicast() {
				continue
			}

			ips = append(ips, ip)
		}

		if len(ips) > 1 {
			log.Info("found machine with more than one IP: %v", ips)
		}

		// populate the endpoint
		if len(ips) > 0 {
			// figure out which network this belongs on
			newip, n := findNet(dc, ips[0])
			if n != nil {
				e, err = dc.Connect(n.ID(), e.ID(), discovery.EDGE_NONE)
				if err != nil {
					log.Fatalln(err)
				}

				edge := e.Edges[len(e.Edges)-1]
				edge.D["ip"] = newip
				edge.D["mac"] = h.MACs[0]
			} else {
				edge := e.NewEdge()
				edge.N = minigraph.UNCONNECTED
				//edge.D["ip"] = ips[0].String()
				edge.D["mac"] = h.MACs[0]
			}
		} else {
			edge := e.NewEdge()
			edge.N = minigraph.UNCONNECTED
			edge.D["mac"] = h.MACs[0]
		}

		max := 0.0
		for k, v := range h.OS {
			if v > max {
				e.D["os"] = k
				max = v
			}
		}

		for _, v := range h.Nameservers {
			ip := net.ParseIP(v).To4()
			if ip == nil || ip.IsLinkLocalUnicast() {
				continue
			}

			if v, ok := e.D["nameserver"]; ok {
				e.D["nameserver"] = fmt.Sprintf("%v,%v", v, ip)
			} else {
				e.D["nameserver"] = ip.String()
			}
		}

		for _, v := range h.Hostnames {
			if strings.Contains(v.Name, ".local") {
				continue
			}

			if names, ok := e.D["hostname"]; ok {
				e.D["hostname"] = fmt.Sprintf("%v,%v", names, v.Name)
			} else {
				e.D["hostname"] = v.Name
			}
		}

		for _, v := range h.Services {
			if ports, ok := e.D["ports"]; ok {
				e.D["ports"] = fmt.Sprintf("%v,%v", ports, v.Port)
			} else {
				e.D["ports"] = strconv.Itoa(int(v.Port))
			}
		}

		if len(h.AdvertisedServices) > 0 {
			b, err := json.Marshal(h.AdvertisedServices)
			if err != nil {
				log.Error("unable to encode advertised services: %v", err)
			} else {
				e.D["advertised_services"] = string(b)
			}
		}

		_, err = dc.UpdateEndpoints(e)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func findNet(dc *discovery.Client, ip net.IP) (string, *minigraph.Network) {
	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		log.Fatalln(err)
	}

	for _, e := range endpoints {
		for _, edg := range e.Edges {
			if dip, ok := edg.D["ip"]; ok {
				_, ipn, err := net.ParseCIDR(dip)
				if err != nil {
					log.Errorln(err)
					continue
				}
				if ipn.Contains(ip) {
					newip := &net.IPNet{
						IP:   ip,
						Mask: ipn.Mask,
					}
					ns, err := dc.GetNetworks("nid", fmt.Sprintf("%v", edg.N))
					if err != nil {
						log.Fatalln(err)
					}
					n := ns[0]
					return newip.String(), n
				}
			}
		}
	}
	return "", nil
}
