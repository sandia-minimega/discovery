// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"net"
	"strconv"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

func AddInterface(dc *discovery.Client, ID int, desc, ip, ip6 string) error {
	if *f_dryrun {
		return nil
	}

	if desc == "" || (ip == "" && ip6 == "") {
		// what's the point?
		return nil
	}

	// try to find network with a matching subnet before creating a new network
	var network *minigraph.Network

	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		log.Fatalln(err)
	}

	var ipnet, ipnet6 *net.IPNet

	if ip != "" {
		var err error
		_, ipnet, err = net.ParseCIDR(ip)
		if err != nil {
			return err
		}
	}

	if ip6 != "" {
		var err error
		_, ipnet6, err = net.ParseCIDR(ip6)
		if err != nil {
			return err
		}
	}

	for _, e := range endpoints {
		for _, edge := range e.Edges {
			if v, ok := edge.D["ip"]; ok {
				_, got, err := net.ParseCIDR(v)
				if ipnet != nil && err == nil && got.String() == ipnet.String() {
					if e.ID() == ID {
						// we found ourselves... redundant info in configs?
						return nil
					}

					// Ayyyy, we found a network this should belong to
					if network != nil && network.ID() != edge.N {
						// oh no
						log.Warn("subnets are wack")
					}

					networks, err := dc.GetNetworks("nid", strconv.Itoa(edge.N))
					if err != nil {
						return err
					}
					if len(networks) == 1 {
						network = networks[0]
					}
				}
			}

			if v, ok := edge.D["ip6"]; ok {
				_, got, err := net.ParseCIDR(v)
				if ipnet6 != nil && err == nil && got.String() == ipnet6.String() {
					if e.ID() == ID {
						// we found ourselves... redundant info in configs?
						return nil
					}

					// Ayyyy, we found a network this should belong to
					if network != nil && network.ID() != edge.N {
						// oh no
						log.Warn("subnets are wack")
					}

					networks, err := dc.GetNetworks("nid", strconv.Itoa(edge.N))
					if err != nil {
						return err
					}
					if len(networks) == 1 {
						network = networks[0]
					}

				}
			}
		}
	}

	// didn't find anything to connect to
	if network == nil {
		n := &minigraph.Network{}
		networks, err := dc.InsertNetworks(n)
		if err != nil {
			return err
		}

		network = networks[0]
	}

	log.Info("connect %v <-> %v -- %v, %v", network.ID(), ID, ip, ip6)
	e, err := dc.Connect(network.ID(), ID, discovery.EDGE_NONE)
	if err != nil {
		return err
	}

	edge := e.Edges[len(e.Edges)-1]
	edge.D["name"] = desc
	if ip != "" {
		edge.D["ip"] = ip
	}
	if ip6 != "" {
		edge.D["ip6"] = ip6
	}

	_, err = dc.UpdateEndpoints(e)
	return err
}
