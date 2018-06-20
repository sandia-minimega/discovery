// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package minigraph

import (
	"fmt"
	"strings"
)

// A network endpoint (regular endpoint, router, etc.). Endpoints contain a map
// for specifying arbitrary information (tags, disk images, etc.).
type Endpoint struct {
	NID   int
	Edges []*Edge
	D     map[string]string
}

func (e *Endpoint) Data() map[string]string {
	return e.D
}

func (e *Endpoint) setID(id int) {
	e.NID = id
}

func (e *Endpoint) Match(k, v string) bool {
	if k != "" {
		if k == "nid" {
			if fmt.Sprintf("%v", e.NID) == v {
				return true
			}
		}

		if val, ok := e.D[k]; ok {
			if strings.Contains(val, v) {
				return true
			}
		}
	} else {
		if fmt.Sprintf("%v", e.NID) == v {
			return true
		}
		for _, val := range e.D {
			if strings.Contains(val, v) {
				return true
			}
		}
	}

	// try the edges
	for _, edge := range e.Edges {
		if edge.Match(k, v) {
			return true
		}
	}

	return false
}

func (e *Endpoint) String() string {
	return fmt.Sprintf("%v", e.ID())
}

// Return the ID of the endpoint.
func (e *Endpoint) ID() int {
	return e.NID
}

// Return the type. Used as a fast alternative to reflection when using the
// Node interface.
func (e *Endpoint) Type() int {
	return TYPE_ENDPOINT
}

// HasEdge returns true when the specified edge is in the endpoint.
func (e *Endpoint) HasEdge(edge *Edge) bool {
	for _, v := range e.Edges {
		if v == edge {
			return true
		}
	}
	return false
}

// Connected returns true if the endpoint is connected to the node referenced
// by the given node ID.
func (e *Endpoint) Connected(n int) bool {
	for _, v := range e.Edges {
		if v.N == n {
			return true
		}
	}
	return false
}

func (e *Endpoint) NewEdge() *Edge {
	edge := newEdge()
	e.Edges = append(e.Edges, edge)
	return edge
}

// Return IDs of connected neighbors
func (e *Endpoint) Neighbors() []int {
	var ret []int
	for _, v := range e.Edges {
		if v.N != UNCONNECTED {
			ret = append(ret, v.N)
		}
	}
	return ret
}
