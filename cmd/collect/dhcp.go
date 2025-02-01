// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"encoding/json"
	"flag"
	"net"
	"strconv"

	"github.com/sandia-minimega/discovery/v2/pkg/commands"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

type CommandDHCP struct {
	commands.Base // embed

	dns string
}

func init() {
	base := commands.Base{
		Flags: flag.NewFlagSet("dns", flag.ExitOnError),
		Usage: "dhcp [OPTION]... <ROUTER ID>",
		Short: "collect MAC->IP entries for DHCP",
		Long: `
Create a MAC->IP mapping for all nodes connected to each edge of the router.
This node can then serve static DHCP addresses.
`,
	}

	cmd := &CommandDHCP{Base: base}
	cmd.Flags.StringVar(&cmd.dns, "dns", "", "IP address of DNS server to advertise")

	commands.Append(cmd)
}

func (c *CommandDHCP) Run() error {
	if c.Flags.NArg() != 1 {
		c.PrintUsage()
		return nil
	}

	router := c.Flags.Arg(0)
	log.Debug("using router id: %v", router)

	// get the router
	rtr, err := dc.GetEndpoint("nid", router)
	if err != nil {
		return err
	}

	for _, edg := range rtr.Edges {
		var assignments = make(map[string]string)

		// for every other endpoint on this network, populate static dhcp
		ns, err := dc.GetNetworks("nid", strconv.Itoa(edg.N))
		if err != nil {
			log.Fatalln(err)
		}
		n := ns[0]

		for _, eid := range n.Endpoints {
			es, err := dc.GetEndpoints("nid", strconv.Itoa(eid))
			if err != nil {
				log.Fatalln(err)
			}
			e := es[0]

			if e.NID == rtr.NID {
				continue
			}

			// find the edge on this endpoint with the n.NID
			for _, eedg := range e.Edges {
				if eedg.N == n.NID {
					if ip, ok := eedg.D["ip"]; ok {
						if mac, ok := eedg.D["mac"]; ok {
							i, _, err := net.ParseCIDR(ip)
							if err != nil {
								log.Fatalln(err)
							}
							assignments[mac] = i.String()
						} else {
							log.Warn("ip %v but no mac", ip)
						}
					}
				}
			}
		}

		if len(assignments) != 0 {
			_, ipn, err := net.ParseCIDR(edg.D["ip"])
			if err != nil {
				log.Fatalln(err)
			}
			edg.D["DHCP"] = ipn.IP.String()

			j, err := json.Marshal(assignments)
			if err != nil {
				log.Fatalln(err)
			}
			edg.D["DHCPStatic"] = string(j)
			if c.dns != "" {
				edg.D["DHCPDNS"] = c.dns
			}
		}
	}

	return UpdateEndpoint(rtr)
}
