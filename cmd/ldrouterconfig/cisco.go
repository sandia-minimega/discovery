// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandia-minimega/discovery/v2/pkg/discovery"
	"github.com/sandia-minimega/discovery/v2/pkg/minigraph"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

func parseCisco(f *os.File, dc *discovery.Client) error {
	e := &minigraph.Endpoint{
		D: map[string]string{
			"router": "true",
			"type":   "cisco",
			"name":   filepath.Base(f.Name()),
			"icon":   "router",
		},
	}

	var ID int

	if !*f_dryrun {
		es, err := dc.InsertEndpoints(e)
		if err != nil {
			return err
		}

		ID = es[0].ID()
	}

	// state machine is "interface" -> description,ip address, !,interface
	scanner := bufio.NewScanner(f)
	var desc, ip, ip6 string

	addInterface := func() error {
		if err := AddInterface(dc, ID, desc, ip, ip6); err != nil {
			return err
		}

		desc = ""
		ip = ""
		ip6 = ""

		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		log.Debug("processing line: %v", line)

		switch fields[0] {
		case "interface":
			if err := addInterface(); err != nil {
				return err
			}
			// use the interface name as the initial description
			desc = strings.Join(fields[1:], " ")
		case "description":
			desc = strings.Join(fields[1:], " ")
		case "ip", "ipv4":
			if len(fields) != 4 || fields[1] != "address" {
				continue
			}
			ipn := &net.IPNet{
				IP:   net.ParseIP(fields[2]),
				Mask: net.IPMask(net.ParseIP(fields[3])),
			}
			log.Debug("got ipv4 address: %v", ipn)
			ip = ipn.String()
		case "ipv6":
			if len(fields) != 3 || fields[1] != "address" {
				continue
			}

			if strings.Contains(line, "link-local") {
				continue
			}

			ip6 = fields[2]
			log.Debug("got ipv6 address: %v", ip6)
		case "bgp":
			if len(fields) != 3 || fields[1] != "router-id" {
				continue
			}

			// found router-id, flush any previous interface
			if err := addInterface(); err != nil {
				return err
			}

			desc = "router-id"
			ip = fields[2] + "/32"
			log.Debug("got router-id: %v", ip)
		}
	}

	if err := addInterface(); err != nil {
		return err
	}

	return scanner.Err()
}
