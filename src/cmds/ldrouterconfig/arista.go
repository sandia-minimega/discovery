// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

func parseArista(f *os.File, dc *discovery.Client) error {
	e := &minigraph.Endpoint{
		D: map[string]string{
			"router": "true",
			"type":   "arista",
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

	// state machine is "interface" -> description,ip[v6] address, !,interface
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
		case "description":
			desc = strings.Join(fields[1:], " ")
		case "ip", "ipv6":
			if len(fields) != 3 || fields[1] != "address" {
				continue
			}

			if fields[0] == "ip" {
				ip = fields[2]
				log.Debug("got ip address: %v", ip)
			} else if !strings.Contains(line, "link-local") {
				ip6 = fields[2]
				log.Debug("got ipv6 address: %v", ip6)
			}
		case "router-id":
			// found router-id, flush any previous interface
			if err := addInterface(); err != nil {
				return err
			}

			desc = "router-id"
			ip = fields[1] + "/32"
			log.Debug("got router-id: %v", ip)
		}
	}

	if err := addInterface(); err != nil {
		return err
	}

	return scanner.Err()
}
