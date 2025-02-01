// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/sandia-minimega/discovery/v2/pkg/discovery"
	"github.com/sandia-minimega/discovery/v2/pkg/minigraph"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

var (
	f_server = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
	f_out    = flag.String("out", "", "save copy of input data for offline processing")
	f_url    = flag.String("url", "https://oscars.es.net/topology-publisher", "URL to process")
)

func usage() {
	fmt.Printf("USAGE: %v [OPTIONS] [FILE]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	log.Init()

	var r io.Reader

	switch flag.NArg() {
	case 0:
		// read from URL
		resp, err := http.Get(*f_url)
		if err != nil {
			log.Fatal("unable to fetch: %v", err)
		}
		defer resp.Body.Close()

		if *f_out != "" {
			// create a file and write all data read from resp.Body to that
			// file so that we have an offline copy
			f, err := os.Create(*f_out)
			if err != nil {
				log.Fatal("unable to create ouput file: %v", err)
			}
			defer f.Close()

			r = io.TeeReader(resp.Body, f)
		} else {
			r = resp.Body
		}
	case 1:
		// read from file
		if *f_out != "" {
			log.Info("igoring -out flag, reading from local file")
		}

		f, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal("unable to open: %v", err)
		}
		defer f.Close()

		r = f
	default:
		usage()
	}

	data := &Data{}
	if err := json.NewDecoder(r).Decode(data); err != nil {
		log.Fatal("unable to decode: %v", err)
	}

	if data.Status == "error" {
		log.Fatal("unable to grab topology: %v", data.Message)
	}

	c := Client{discovery.New(*f_server)}

	// keep track of link IDs -> Network ID
	links := map[string]int{}
	// keep track of the networks that we create
	networks := map[int]*minigraph.Network{}

	for _, d := range data.Domains {
		// TODO: figure out the difference between the domains
		if !strings.Contains(d.ID, "ps.es.net") {
			continue
		}

		for _, n := range d.Nodes {
			log.Info("found node %v at %v %v", n.ID, n.Latitude, n.Longitude)

			e := c.newEndpoint()
			e.D = map[string]string{
				"name":      n.Name,
				"urn":       n.ID,
				"hostname":  n.Hostname,
				"latitude":  n.Latitude,
				"longitude": n.Longitude,
				"router":    "true",
			}

			for _, p := range n.Ports {
				// if a port has an IP, then it is a peering point
				if p.IPAddress != "" {
					network := c.newNetwork()
					network.Endpoints = append(network.Endpoints, e.ID())

					e.Edges = append(e.Edges, &minigraph.Edge{
						N: network.ID(),
						D: map[string]string{
							"ip":          p.IPAddress + "/" + p.Netmask,
							"description": p.Description,
							"capacity":    p.Capacity,
							"OSPF":        "true",
						},
					})

					networks[network.ID()] = network
				}

				// if a port has links, there is a link between esnet routers
				for _, l := range p.Links {
					if l.Name == "" {
						continue
					}
					if l.NameType != "logical" {
						log.Info("expected logical nameType, not %v", l.NameType)
						continue
					}

					var network *minigraph.Network

					// check to see if we already have a network assigned for
					// this (i.e. we have already processed the other end of
					// this link)
					if nid, ok := links[l.ID]; !ok {
						network = c.newNetwork()
						networks[network.ID()] = network

						// we never reprocess l.ID so omit that from links
						links[l.RemoteLinkID] = network.ID()
					} else {
						network = networks[nid]
					}

					e.Edges = append(e.Edges, &minigraph.Edge{
						N: network.ID(),
						D: map[string]string{
							"ip":          l.Name,
							"description": p.Description,
							"capacity":    p.Capacity,
							"OSPF":        "true",
						},
					})

					network.Endpoints = append(network.Endpoints, e.ID())
				}
			}

			// push the updated endpoint
			if _, err := c.UpdateEndpoints(e); err != nil {
				log.Fatal("unable to update endpoint: %v", err)
			}
		}
	}

	// push the networks
	c.pushNetworks(networks)
}
