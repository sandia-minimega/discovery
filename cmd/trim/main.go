// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/sandia-minimega/discovery/v2/pkg/discovery"
	"github.com/sandia-minimega/discovery/v2/pkg/minigraph"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

var (
	f_server      = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
	f_unconnected = flag.Bool("unconnected", false, "trim nodes that are not connected to other nodes")
	f_delete      = flag.Bool("delete", false, "trim nodes by deleting them (default is to mark them trimmed=true)")
	f_size        = flag.Int("size", -1, "trim until -size nodes are left")
	f_root        = flag.Int("root", -1, "root node ID for walks")
	f_trimmed     = flag.Bool("trimmed", false, "trim trimmed=true endpoints (use with -delete)")
	f_clear       = flag.Bool("clear", false, "clear trimmed flags on endpoints")
	f_search      = flag.String("q", "", "trim nodes matching term or key=value")
)

type Client struct {
	*discovery.Client // embed
}

func (c Client) trimNode(id int) error {
	log.Info("trimming: %v", id)

	if *f_delete {
		_, err := c.DeleteEndpoints("nid", strconv.Itoa(id))
		return err
	}

	es, err := c.GetEndpoints("nid", strconv.Itoa(id))
	if err != nil {
		return err
	}

	if len(es) != 1 {
		return fmt.Errorf("expected 1 endpoint for ID %v, not %v", id, len(es))
	}

	e := es[0]

	e.D["trimmed"] = "true"

	_, err = c.UpdateEndpoints(e)
	return err
}

// getEndpoints wraps discovery.Client.GetEndpoints and turns the result into
// a map based on endpoint ID instead of a slice.
func (c Client) getEndpoints(args ...string) map[int]*minigraph.Endpoint {
	var k, v string

	switch len(args) {
	case 0:
		// ignore
	case 1:
		v = args[0]
	case 2:
		k, v = args[0], args[1]
	default:
		log.Fatal("too many args to getEndpoints: %v", args)
	}

	endpoints, err := c.GetEndpoints(k, v)
	if err != nil {
		log.Fatalln(err)
	}

	res := map[int]*minigraph.Endpoint{}
	for _, n := range endpoints {
		res[n.ID()] = n
	}

	return res
}

// getNetworks wraps discovery.Client.GetNetworks and turns the result into a
// map based on network ID instead of a slice
func (c Client) getNetworks() map[int]*minigraph.Network {
	networks, err := c.GetNetworks("", "")
	if err != nil {
		log.Fatalln(err)
	}

	res := map[int]*minigraph.Network{}
	for _, n := range networks {
		res[n.ID()] = n
	}

	return res
}

// connected walks the graph from the specified node and returns a map of nodes
// that are connected to it.
func (c Client) connected(id int) map[int]bool {
	networks := c.getNetworks()

	// list of networks to process and networks that have been visited
	working := []int{}
	visited := map[int]bool{}

	// mapping of endpoint -> list of networks
	reverse := map[int][]int{}

	// find the initial starting point
	for _, n := range networks {
		for _, v := range n.Endpoints {
			if v == id {
				log.Debug("inital network: %v", n.ID())
				working = append(working, n.ID())
				visited[n.ID()] = true
			}

			reverse[v] = append(reverse[v], n.ID())
		}
	}

	// expand visited until we run out of new networks to visit
	for len(working) > 0 {
		next := []int{}

		for _, n := range working {
			log.Debug("visiting %v", n)

			for _, e := range networks[n].Endpoints {
				for _, n2 := range reverse[e] {
					if !visited[n2] {
						visited[n2] = true
						next = append(next, n2)
					}
				}
			}
		}

		working = next
	}

	// build result -- visited map for endpoints
	res := map[int]bool{}

	// go through and mark endpoints as visited
	for e, networks := range reverse {
		for _, n := range networks {
			if visited[n] {
				res[e] = true
				break
			}
		}
	}

	return res
}

// trimDisconnected trims endpoints with no edges or edges that are all
// disconnected.
func (c Client) trimDisconnected() {
	endpoints := c.getEndpoints()

Outer:
	for _, endpoint := range endpoints {
		for _, edge := range endpoint.Edges {
			if edge.N != minigraph.UNCONNECTED {
				// found a connection -- go to the next endpoint
				continue Outer
			}
		}

		if err := c.trimNode(endpoint.ID()); err != nil {
			log.Error("trim %v: %v", endpoint, err)
		}
	}
}

// trimUnconnected trims endpoints that are not connected to the specified
// node.
func (c Client) trimUnconnected(id int) {
	endpoints := c.getEndpoints()

	connected := c.connected(id)

	for id := range endpoints {
		if !connected[id] {
			if err := c.trimNode(id); err != nil {
				log.Error("trim %v: %v", id, err)
			}
		}
	}
}

// trimSize trims endpoints until we reach the desired size
func (c Client) trimSize(root, size int) {
	endpoints := c.getEndpoints()

	connected := c.connected(root)

	// start with unconnected... maybe we'll trim enough
	for id := range endpoints {
		if !connected[id] {
			if err := c.trimNode(id); err != nil {
				log.Error("trim %v: %v", id, err)
			}

			delete(endpoints, id)
			if len(endpoints) == size {
				return
			}
		}
	}

	// need to trim more than just unconnected endpoints, start trimming based
	// on the number of networks endpoints are connect to (in ascending order)
	sorted := []int{}
	for k := range endpoints {
		sorted = append(sorted, k)
	}

	sort.Slice(sorted, func(i, j int) bool {
		e := endpoints[sorted[i]]
		e2 := endpoints[sorted[j]]

		// if they have the same number of edges, sort on NID instead
		if len(e.Edges) == len(e2.Edges) {
			return e.NID < e2.NID
		}
		return len(e.Edges) < len(e2.Edges)
	})

	for len(endpoints) > size && len(sorted) > 0 {
		id := sorted[0]
		sorted = sorted[1:]

		if id == root {
			log.Warn("trimming the root!")
		}

		// trim the next node from sorted
		if err := c.trimNode(id); err != nil {
			log.Error("trim %v: %v", id, err)
		}

		delete(endpoints, id)
	}
}

// trimQuery trims endpoints matching the query
func (c Client) trimQuery(q string) {
	endpoints := c.getEndpoints(strings.SplitN(q, "=", 2)...)

	// trim all matches
	for id := range endpoints {
		if err := c.trimNode(id); err != nil {
			log.Error("trim %v: %v", id, err)
		}
	}
}

// trimTrimmed trims endpoints with trimmed=true
func (c Client) trimTrimmed() {
	endpoints := c.getEndpoints()

	for id, e := range endpoints {
		if e.D["trimmed"] == "true" {
			if err := c.trimNode(id); err != nil {
				log.Error("trim %v: %v", id, err)
			}
		}
	}
}

// clearTrim clears trimmed=true flag from endpoints
func (c Client) clearTrim() {
	endpoints := c.getEndpoints()

	for id, e := range endpoints {
		if e.D["trimmed"] == "true" {
			delete(e.D, "trimmed")

			_, err := c.UpdateEndpoints(e)
			if err != nil {
				log.Fatalln("clear trim on %v: %v", id, err)
			}
		}
	}
}

func usage() {
	fmt.Printf("USAGE: %v [OPTIONS]", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	log.Init()

	c := Client{discovery.New(*f_server)}

	if *f_clear {
		log.Infoln("clearing trimmed flag on nodes")
		c.clearTrim()
		return
	}

	if *f_size > -1 {
		if *f_root == -1 {
			log.Fatalln("must specify root with -size")
		}

		log.Info("trimming to size %v", *f_size)
		c.trimSize(*f_root, *f_size)
		return
	}

	if *f_unconnected {
		if *f_root == -1 {
			log.Fatalln("must specify root with -unconnected")
		}

		log.Info("trimming nodes not connected to %v", *f_root)
		c.trimUnconnected(*f_root)
		return
	}

	if *f_trimmed {
		if !*f_delete {
			log.Fatalln("no-op without -delete")
		}

		log.Infoln("trimming trimmed nodes")
		c.trimTrimmed()
		return
	}

	if *f_search != "" {
		log.Infoln("trim query")
		c.trimQuery(*f_search)
		return
	}

	log.Infoln("trimming disconnected nodes")
	c.trimDisconnected()
}
