// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"encoding/json"
	"flag"
	"net"
	"strings"

	"github.com/sandia-minimega/discovery/v2/pkg/commands"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

type CommandDNS struct {
	commands.Base // embed

	cidr, domain string
}

func init() {
	base := commands.Base{
		Flags: flag.NewFlagSet("dns", flag.ExitOnError),
		Usage: "dns [OPTION]... <ID>",
		Short: "collect IP->hostname entries for DNS",
		Long: `
Create a IP->hostname mapping for all nodes matching the filters and add the
mapping to the given node. This node can then resolve DNS requests.
`,
	}

	cmd := &CommandDNS{Base: base}
	cmd.Flags.StringVar(&cmd.cidr, "filter-cidr", "", "filter by cidr")
	cmd.Flags.StringVar(&cmd.domain, "filter-domain", "", "filter by domain")

	commands.Append(cmd)
}

func (c *CommandDNS) Run() error {
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

	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		return err
	}

	// default filters don't filter anything
	filterCIDR := func(net.IP) bool {
		return false
	}
	filterDomain := func(string) bool {
		return false
	}

	if c.cidr != "" {
		log.Info("filtering on cidr %v", c.cidr)
		_, ipnet, err := net.ParseCIDR(c.cidr)
		if err != nil {
			log.Fatalln(err)
		}
		filterCIDR = func(v net.IP) bool {
			return !ipnet.Contains(v)
		}
	}

	if c.domain != "" {
		log.Info("filtering on domain %v", c.domain)
		filterDomain = func(v string) bool {
			return !strings.Contains(v, c.domain)
		}
	}

	var resolv = make(map[string]string)

	for _, v := range endpoints {
		ips := GetIPs(v)

		if len(ips) == 0 {
			log.Error("no ip on endpoint: %v", v.NID)
			continue
		}

		// TODO: what should we do if the node has more than one IP? We should
		// probably need to track hostnames on a per-edge basis.

		if len(ips) > 1 {
			log.Info("%v has multiple IPs: %v", v.NID, ips)
		}
		ip := ips[0]

		if filterCIDR(ip) {
			log.Debug("filtering %v based on cidr filter", ip)
			continue
		}

		if hostnames, ok := v.D["hostname"]; ok {
			hosts := strings.Split(hostnames, ",")
			for _, h := range hosts {
				if filterDomain(h) {
					log.Debug("filtering %v based on domain filter", h)
					continue
				}

				resolv[h] = ip.String()
			}
		}
	}

	if len(resolv) > 0 {
		j, err := json.Marshal(resolv)
		if err != nil {
			log.Fatalln(err)
		}
		rtr.D["dns"] = string(j)

		return UpdateEndpoint(rtr)
	}

	return nil
}
