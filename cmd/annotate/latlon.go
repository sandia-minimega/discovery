// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"strconv"
	"strings"

	"github.com/sandia-minimega/discovery/v2/pkg/commands"
	"github.com/sandia-minimega/discovery/v2/pkg/latlon"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

type CommandLatLon struct {
	commands.Base // embed

	dir string
}

func init() {
	base := commands.Base{
		Flags: flag.NewFlagSet("latlon", flag.ExitOnError),
		Usage: "latlon [OPTION]...",
		Short: "annotate with geoip information",
		Long: `
Annotates nodes using a GeoIP database and node IP addresses. Adds latitude,
longitude, country and ISP information to each node.
`,
	}

	cmd := &CommandLatLon{Base: base}
	cmd.Flags.StringVar(&cmd.dir, "dir", "", "directory to scan for geoip files (*.mmdb)")

	commands.Append(cmd)
}

func (c *CommandLatLon) Run() error {
	db, err := latlon.Open(c.dir)
	if err != nil {
		return err
	}
	defer db.Close()

	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		return err
	}

	for _, v := range endpoints {
		ips := GetIPs(v)

		if len(ips) == 0 {
			log.Error("no ip on endpoint: %v", v.NID)
			continue
		}

		// TODO: should we annotate with a location per IP instead of the first
		// IP with a valid location?

		var res *latlon.Result
		for _, ip := range ips {
			res2, err := db.Lookup(ips[0])
			if err != nil {
				log.Error("unable to lookup %v: %v", ip, err)
				continue
			}

			res = res2
			break
		}
		if res == nil {
			continue
		}

		parts := []string{}
		if s := res.City.City.Names["en"]; s != "" {
			parts = append(parts, s)
		}
		for _, s := range res.City.Subdivisions {
			parts = append(parts, s.Names["en"])
		}
		if s := res.City.Country.Names["en"]; s != "" {
			parts = append(parts, s)
		}

		if len(parts) > 0 {
			v.D["geoip_location"] = strings.Join(parts, ", ")
		}

		v.D["geoip_lat"] = strconv.FormatFloat(res.City.Location.Latitude, 'f', -1, 64)
		v.D["geoip_lon"] = strconv.FormatFloat(res.City.Location.Longitude, 'f', -1, 64)

		v.D["geoip_isp"] = res.ISP.ISP

		if s := res.City.Country.IsoCode; s != "" {
			v.D["geoip_country"] = strings.ToUpper(s)
		}

		if err := UpdateEndpoint(v); err != nil {
			return err
		}
	}

	return nil
}
