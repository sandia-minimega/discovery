// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"net"
	"strings"

	"pkg/commands"
	log "pkg/minilog"

	"github.com/google/gopacket/macs"
)

type CommandIcon struct {
	commands.Base // embed
}

func init() {
	base := commands.Base{
		Usage: "icon",
		Short: "annotate with icons based on other fields",
		Long: `
Annotate nodes with icons based on other fields such as hostname, MAC, and
country code.
`,
	}
	commands.Append(&CommandIcon{base})
}

func (c *CommandIcon) Run() error {
	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		return err
	}

	for _, v := range endpoints {
		if _, ok := v.D["icon"]; ok && !*f_overwrite {
			continue
		}

		var icons []string
		if _, ok := v.D["hostname"]; ok {
			if v := iconByName(v.D["hostname"]); v != "" {
				icons = append(icons, v)
			}
		}

		for _, edge := range v.Edges {
			if mac, ok := edge.D["mac"]; ok {
				if v := iconByMAC(mac); v != "" {
					icons = append(icons, v)
				}
			}
		}

		if _, ok := v.D["geoip_country"]; ok {
			icons = append(icons, v.D["geoip_country"])
		}

		if len(icons) == 0 {
			log.Debug("no clue how to icon node %v", v)
			continue
		}

		v.D["icon"] = strings.Join(icons, ",")
		log.Debug("updating node %v with icon: %v", v, v.D["icon"])

		if err := UpdateEndpoint(v); err != nil {
			return err
		}
	}

	return nil
}

func iconByName(name string) string {
	name = strings.ToLower(name)

	switch {
	case strings.Contains(name, "android"):
		return "linux" // TODO: customize?
	case strings.Contains(name, "apple"):
		return "apple"
	case strings.Contains(name, "macbook"):
		return "apple"
	case strings.Contains(name, "mbp"):
		return "apple"
	case strings.Contains(name, "ipod"):
		return "apple" // TODO: customize?
	case strings.Contains(name, "iphone"):
		return "apple" // TODO: customize?
	case strings.Contains(name, "ipad"):
		return "apple" // TODO: customize?
	case strings.Contains(name, "windows"):
		return "windows"
	}

	return ""
}

func iconByMAC(mac string) string {
	v, err := net.ParseMAC(mac)
	if err != nil {
		return ""
	}

	org, ok := macs.ValidMACPrefixMap[[3]byte{v[0], v[1], v[2]}]
	if !ok {
		return ""
	}

	log.Debug("found org: %v", org)

	return iconByName(org)
}
