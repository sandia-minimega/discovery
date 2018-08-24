// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

// Client wraps a discovery.Client with additional helper method
type Client struct {
	*discovery.Client
}

// newEndpoint creates a new endpoint, calling log.Fatal if there's an error
func (c *Client) newEndpoint() *minigraph.Endpoint {
	e := &minigraph.Endpoint{}
	es, err := c.InsertEndpoints(e)
	if err != nil {
		log.Fatal("unable to create endpoint: %v", err)
	}

	if len(es) != 1 {
		log.Fatal("expected 1 endpoint not %v", len(es))
	}

	return es[0]
}

// newNetwork creates a new network, calling log.Fatal if there's an error
func (c *Client) newNetwork() *minigraph.Network {
	n := &minigraph.Network{}
	ns, err := c.InsertNetworks(n)
	if err != nil {
		log.Fatal("unable to create network: %v", err)
	}

	if len(ns) != 1 {
		log.Fatal("expected 1 network not %v", len(ns))
	}

	return ns[0]
}

func (c *Client) pushNetworks(networks map[int]*minigraph.Network) {
	ns := []*minigraph.Network{}
	for _, n := range networks {
		ns = append(ns, n)
	}

	if _, err := c.UpdateNetworks(ns...); err != nil {
		log.Fatal("unable to update networks: %v", err)
	}
}
