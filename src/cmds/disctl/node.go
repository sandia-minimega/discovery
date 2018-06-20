// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

// updating fields is crazy.
//	[endpoint key=]<value> <key> <value>
//	[endpoint key=]<value> [edge key=]<value> <key> <value>
func update() (string, error) {
	args := flag.Args()
	if len(args) != 3 && len(args) != 4 {
		return "", fmt.Errorf("inavlid arguments: %v", args)
	}

	endpointSearch := args[0]
	edgeSearch := args[1]
	key := args[len(args)-2]
	value := args[len(args)-1]

	endpoint, err := findUniqueEndpoint(endpointSearch)
	if err != nil {
		return "", err
	}
	if len(args) == 4 {
		edge, err := findUniqueEdge(endpoint, edgeSearch)
		if err != nil {
			return "", err
		}
		edge.D[key] = value
	} else {
		endpoint.D[key] = value
	}

	ret, err := dc.UpdateEndpoints(endpoint)
	var r string
	for i, v := range ret {
		if i == 0 {
			r = fmt.Sprintf("%v", v)
		} else {
			r = fmt.Sprintf("%v%v", r, v)
		}
		if i != len(ret)-1 {
			r += "\n\n"
		}
	}
	return r, err
}

// find networks based on properties of connected edges
func findNetworks() (string, error) {
	// we expect exactly 1 additional cli
	//	 [node key=]<value>
	var err error

	args := flag.Args()

	if len(args) != 1 {
		return "", fmt.Errorf("invalid arguments: %v", args)
	}

	f := strings.Split(args[0], "=")

	var n []*minigraph.Network
	var e []*minigraph.Endpoint

	var a, b string

	switch len(f) {
	case 1:
		a, b = "", f[0]
	case 2:
		a, b = f[0], f[1]
	default:
		return "", fmt.Errorf("invalid search term: %v", args[0])
	}

	n, err = dc.GetNetworks(a, b)
	e, err = dc.GetEndpoints(a, b)

	res := map[string]bool{}

	for _, v := range n {
		res[v.String()] = true
	}

	for _, v := range e {
		// find network ids of matching edges
		for _, edg := range v.Edges {
			if edg.Match(a, b) {
				n, err = dc.GetNetworks("nid", fmt.Sprintf("%v", edg.N))
				if len(n) != 1 {
					log.Fatal("unexpected network list: %v", n)
				}
				res[n[0].String()] = true
			}
		}
	}

	keys := []string{}
	for k := range res {
		keys = append(keys, k)
	}

	return strings.Join(keys, "\n\n"), err
}

func find() (string, error) {
	// we expect exactly 1 additional cli
	//	 [node key=]<value>
	var err error

	args := flag.Args()

	if len(args) != 1 {
		return "", fmt.Errorf("invalid arguments: %v", args)
	}

	f := strings.Split(args[0], "=")

	var n []*minigraph.Network
	var e []*minigraph.Endpoint

	switch len(f) {
	case 1:
		n, err = dc.GetNetworks("", f[0])
		e, err = dc.GetEndpoints("", f[0])
	case 2:
		n, err = dc.GetNetworks(f[0], f[1])
		e, err = dc.GetEndpoints(f[0], f[1])
	default:
		return "", fmt.Errorf("invalid search term: %v", args[0])
	}

	var r string
	for _, v := range n {
		if r == "" {
			r = fmt.Sprintf("%v", v)
		} else {
			r = fmt.Sprintf("%v%v", r, v)
		}
	}
	for i, v := range e {
		if r == "" {
			r = fmt.Sprintf("%v", v)
		} else {
			r = fmt.Sprintf("%v%v", r, v)
		}
		if i != len(e)-1 {
			r += "\n\n"
		}
	}
	return r, err
}

func remove() (string, error) {
	// we expect exactly 1 additional cli
	//	 [node key=]<value>
	var err error

	args := flag.Args()

	if len(args) != 1 {
		return "", fmt.Errorf("invalid arguments: %v", args)
	}

	e, err := findUniqueEndpoint(args[0])
	if err == nil {
		ret, err := dc.DeleteEndpoints("NID", fmt.Sprintf("%v", e.NID))
		return fmt.Sprintf("%v", ret), err
	}
	n, err := findUniqueNetwork(args[0])
	if err == nil {
		ret, err := dc.DeleteNetworks("NID", fmt.Sprintf("%v", n.NID))
		return fmt.Sprintf("%v", ret), err
	}

	return "", err
}

func endpointInsert() (string, error) {
	n := &minigraph.Endpoint{}
	ret, err := dc.InsertEndpoints(n)

	var r string
	for i, v := range ret {
		if i == 0 {
			r = fmt.Sprintf("%v", v)
		} else {
			r = fmt.Sprintf("%v%v", r, v)
		}
		if i != len(ret)-1 {
			r += "\n\n"
		}
	}
	return r, err
}

func networkInsert() (string, error) {
	n := &minigraph.Network{}
	ret, err := dc.InsertNetworks(n)

	var r string
	for i, v := range ret {
		if i == 0 {
			r = fmt.Sprintf("%v", v)
		} else {
			r = fmt.Sprintf("%v%v", r, v)
		}
		if i != len(ret)-1 {
			r += "\n\n"
		}
	}
	return r, err
}

func connect() (string, error) {
	// we expect exactly 2 or 3 additional CLI args:
	//	 [network key=]<value> [endpoint key=]<value> [edge index]
	var err error

	args := flag.Args()

	if len(args) != 2 && len(args) != 3 {
		return "", fmt.Errorf("invalid arguments: %v", args)
	}

	n, err := findUniqueNetwork(args[0])
	if err != nil {
		return "", err
	}

	e, err := findUniqueEndpoint(args[1])
	if err != nil {
		return "", err
	}

	var eidx int
	if len(args) == 3 {
		eidx, err = strconv.Atoi(args[2])
		if err != nil {
			return "", err
		}
	} else {
		eidx = discovery.EDGE_NONE
	}

	ret, err := dc.Connect(n.NID, e.NID, eidx)
	return fmt.Sprintf("%v", ret), err
}

func disconnect() (string, error) {
	// we expect exactly 2 additional cli args
	//	 [network key=]<value> [endpoint key=]<value>
	var err error

	args := flag.Args()

	if len(args) != 2 {
		return "", fmt.Errorf("invalid arguments: %v", args)
	}

	n, err := findUniqueNetwork(args[0])
	if err != nil {
		return "", err
	}

	e, err := findUniqueEndpoint(args[1])
	if err != nil {
		return "", err
	}

	ret, err := dc.Disconnect(n.NID, e.NID)
	return fmt.Sprintf("%v", ret), err
}

func findUniqueNetwork(s string) (*minigraph.Network, error) {
	f := strings.Split(s, "=")

	var n []*minigraph.Network
	var err error

	switch len(f) {
	case 1:
		n, err = dc.GetNetworks("", f[0])
	case 2:
		n, err = dc.GetNetworks(f[0], f[1])
	default:
		return nil, fmt.Errorf("invalid search term: %v", s)
	}

	if err != nil {
		return nil, err
	}

	if len(n) == 0 {
		return nil, fmt.Errorf("no networks found")
	}

	if len(n) != 1 {
		return nil, fmt.Errorf("search yielded multiple results: %v", n)
	}

	return n[0], nil
}

func findUniqueEndpoint(s string) (*minigraph.Endpoint, error) {
	log.Debug("got search term: %v", s)

	f := strings.Split(s, "=")

	var e []*minigraph.Endpoint
	var err error

	switch len(f) {
	case 1:
		e, err = dc.GetEndpoints("", f[0])
	case 2:
		e, err = dc.GetEndpoints(f[0], f[1])
	default:
		return nil, fmt.Errorf("invalid search term: %v", s)
	}

	if err != nil {
		return nil, err
	}

	if len(e) == 0 {
		return nil, fmt.Errorf("no endpoints found")
	}

	if len(e) != 1 {
		return nil, fmt.Errorf("search yielded multiple results: %v", e)
	}

	return e[0], nil
}

func findUniqueEdge(e *minigraph.Endpoint, s string) (*minigraph.Edge, error) {
	log.Debug("got edge search term: %v", s)

	f := strings.Split(s, "=")

	var edge *minigraph.Edge

	var key, value string

	switch len(f) {
	case 1:
		value = s
	case 2:
		key = f[0]
		value = f[1]
	default:
		return nil, fmt.Errorf("invalid search term: %v", s)
	}

	for _, v := range e.Edges {
		if v.Match(key, value) {
			if edge != nil {
				return nil, fmt.Errorf("search yielded multiple results: %v", e)
			}
			edge = v
		}
	}

	if edge == nil {
		return nil, fmt.Errorf("no endpoints found")
	}

	return edge, nil
}
