// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

type Updater struct {
	*discovery.Client
	masks map[int][]*net.IPNet
	count int
}

var (
	f_server = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
	f_limit  = flag.Int("limit", 1000, "limit the number of clients to add")
	f_start  = flag.String("start", "", "earliest time to add")
	f_end    = flag.String("end", "", "latest time to add")
)

func usage() {
	fmt.Printf("USAGE: %v [OPTIONS] FILE\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Parse()

	log.Init()

	if flag.NArg() != 1 {
		usage()
	}

	var start, end time.Time

	if *f_start != "" {
		v, err := time.Parse("Jan 02 15:04:05", *f_start)
		if err != nil {
			log.Fatal("invalid start time: %v", err)
		}
		start = v
	}
	if *f_end != "" {
		v, err := time.Parse("Jan 02 15:04:05", *f_end)
		if err != nil {
			log.Fatal("invalid end time: %v", err)
		}
		end = v
	}

	filename := flag.Arg(0)
	log.Debug("using filename: %v", filename)

	f, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	u := &Updater{
		Client: discovery.New(*f_server),
	}
	u.PopulateNetmasks()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if u.count > *f_limit {
			break
		}

		line := scanner.Text()

		if !strings.Contains(line, "DHCPACK") {
			continue
		}

		fields := strings.Fields(line)

		// first three fields should be the timestamp of the ACK
		t, err := time.Parse("Jan 02 15:04:05", strings.Join(fields[:3], " "))
		if err != nil {
			log.Error("unable to parse time: %v", err)
			continue
		}

		if !start.IsZero() && t.Before(start) {
			continue
		}
		if !end.IsZero() && t.After(end) {
			// assume that the file is in order...
			break
		}

		var i int
		for i = 0; i < len(fields); i++ {
			if fields[i] == "DHCPACK" {
				break
			}
		}
		if i == len(fields) {
			// that's strange, didn't find the DHCPACK after all
			continue
		}
		// trim unused fields
		fields = fields[i:]

		if len(fields) < 2 {
			continue
		}

		var ip, mac, hostname string

		switch fields[1] {
		case "on":
			// Example:
			//   DHCPACK on 10.221.X.Y to yy:yy:yy:yy:yy:yy via 10.221.X.1
			//   DHCPACK on 10.221.X.Y to yy:yy:yy:yy:yy:yy (Y) via 10.221.X.1
			if len(fields) < 7 {
				continue
			}

			ip = fields[2]
			mac = fields[4]

			if fields[5] != "via" {
				hostname = strings.Trim(fields[5], "()")
			}

		case "to":
			// Example:
			//   DHCPACK to 10.221.X.Z (zz:zz:zz:zz:zz:zz) via em1
			if len(fields) != 6 {
				continue
			}

			ip = fields[2]
			mac = strings.Trim(fields[3], "()")
		}

		log.Debug("ip: %v, mac: %v, hostname: %v", ip, mac, hostname)

		e, err := u.GetOrCreate(mac)
		if err != nil {
			log.Fatalln(err)
		}

		if hostname != "" {
			e.D["name"] = hostname
		}

		if _, err := u.UpdateEndpoints(e); err != nil {
			log.Fatalln(err)
		}

		if err := u.Update(e, ip, mac); err != nil {
			log.Fatalln(err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalln(err)
	}
}

func (u *Updater) PopulateNetmasks() {
	u.masks = map[int][]*net.IPNet{}

	endpoints, err := u.GetEndpoints("", "")
	if err != nil {
		log.Fatalln(err)
	}

	for _, e := range endpoints {
		for _, edge := range e.Edges {
			for _, v := range []string{"ip", "ip6"} {
				if ip, ok := edge.D[v]; ok {
					_, ipn, err := net.ParseCIDR(ip)
					if err != nil {
						log.Errorln(err)
						continue
					}

					u.masks[edge.N] = append(u.masks[edge.N], ipn)
				}
			}
		}
	}
}

func (u *Updater) GetOrCreate(mac string) (*minigraph.Endpoint, error) {
	endpoints, err := u.GetEndpoints("mac", mac)
	if err != nil {
		return nil, err
	}

	if len(endpoints) > 1 {
		log.Info("more than one endpoint with MAC: %v", mac)
		return endpoints[0], nil
	} else if len(endpoints) == 1 {
		return endpoints[0], nil
	}

	// create a new endpoint
	e := &minigraph.Endpoint{}
	es, err := u.InsertEndpoints(e)
	if err != nil {
		return nil, err
	}

	return es[0], nil
}

func (u *Updater) Update(e *minigraph.Endpoint, s, mac string) error {
	ip := net.ParseIP(s)
	if ip == nil {
		// complain but don't kill everything
		log.Error("invalid IP: %v", s)
		return nil
	}

	for id, subnets := range u.masks {
		for _, subnet := range subnets {
			if subnet.Contains(ip) {
				index := discovery.EDGE_NONE

				// check to see what edges the node already has and reconnect
				// it in a different place if it already has an edge with the
				// same MAC
				for i, edge := range e.Edges {
					if edge.D["mac"] == mac {
						if id == edge.N {
							// already connected in the right place
							return nil
						}

						// fuck it
						return nil

						if _, err := u.Disconnect(e.ID(), edge.N); err != nil {
							return err
						}

						index = i
					}
				}

				e, err := u.Connect(id, e.ID(), index)
				if err != nil {
					return err
				}

				if index == discovery.EDGE_NONE {
					index = len(e.Edges) - 1
				}

				edge := e.Edges[index]
				edge.D["ip"] = (&net.IPNet{
					IP:   ip,
					Mask: subnet.Mask,
				}).String()
				edge.D["mac"] = mac

				u.count += 1
				_, err = u.UpdateEndpoints(e)
				return err
			}
		}
	}

	log.Info("unable to find subnet for %v", ip)

	return nil
}
