// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package minigraph

import (
	"encoding/gob"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	log "pkg/minilog"
)

const (
	TYPE_NODE = iota
	TYPE_ENDPOINT
	TYPE_NETWORK
)

const (
	UNCONNECTED = -1
)

type Graph struct {
	lock  sync.Mutex
	Nodes map[int]Node
	maxID int
}

type Node interface {
	ID() int
	Type() int
	Connected(int) bool
	Neighbors() []int
	Match(string, string) bool
	setID(int)
	Data() map[string]string
}

type networks []*Network

func (n networks) Len() int           { return len(n) }
func (n networks) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n networks) Less(i, j int) bool { return n[i].ID() < n[j].ID() }

type endpoints []*Endpoint

func (n endpoints) Len() int           { return len(n) }
func (n endpoints) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n endpoints) Less(i, j int) bool { return n[i].ID() < n[j].ID() }

func init() {
	gob.Register(&Endpoint{})
	gob.Register(&Network{})
	gob.Register(Edge{})
}

func New() *Graph {
	return &Graph{
		Nodes: make(map[int]Node),
	}
}

// read graph contents from an io.Reader. If an io.EOF occurs without reading
// any data, assume an empty reader and return a new graph.
func Read(f io.Reader) (*Graph, error) {
	g := New()
	g.lock.Lock()
	defer g.lock.Unlock()
	dec := gob.NewDecoder(f)
	err := dec.Decode(g)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return g, nil
}

func (g *Graph) Write(f io.Writer) error {
	g.lock.Lock()
	defer g.lock.Unlock()
	enc := gob.NewEncoder(f)
	err := enc.Encode(g)
	if err != nil {
		return err
	}
	return nil
}

func (g *Graph) NewEndpoint() *Endpoint {
	n := &Endpoint{
		NID: g.newID(),
		D:   make(map[string]string),
	}
	g.Nodes[n.ID()] = n
	return n
}

func (g *Graph) NewNetwork() *Network {
	n := &Network{
		NID: g.newID(),
		D:   make(map[string]string),
	}
	g.Nodes[n.ID()] = n
	return n
}

func (g *Graph) newID() int {
	g.lock.Lock()
	defer g.lock.Unlock()
	if g.maxID == 0 {
		for k, _ := range g.Nodes {
			if g.maxID < k {
				g.maxID = k
			}
		}
	}
	g.maxID++
	log.Debug("new id: %v", g.maxID)
	return g.maxID
}

func (g *Graph) Insert(n Node) (Node, error) {
	// add an ID if the current ID is 0
	if n.ID() == 0 {
		n.setID(g.newID())
	}
	if _, ok := g.Nodes[n.ID()]; !ok {
		g.Nodes[n.ID()] = n
		return n, nil
	}
	return n, fmt.Errorf("node %v already exists", n)
}

// Delete a node from the graph and remove all references to it in other nodes.
func (g *Graph) Delete(n Node) error {
	if _, ok := g.Nodes[n.ID()]; ok {
		for _, v := range n.Neighbors() {
			err := g.Disconnect(n, g.Nodes[v])
			if err != nil {
				return err
			}
		}
		delete(g.Nodes, n.ID())
		return nil
	}
	return fmt.Errorf("no such node %v", n)
}

func (g *Graph) GetNodes() []Node {
	var ret []Node
	for _, v := range g.Nodes {
		ret = append(ret, v)
	}
	return ret
}

func (g *Graph) GetEndpoints() []*Endpoint {
	var ret []*Endpoint
	for _, v := range g.Nodes {
		switch v.(type) {
		case *Endpoint:
			ret = append(ret, v.(*Endpoint))
		}
	}
	return ret
}

func (g *Graph) GetNetworks() []*Network {
	var ret []*Network
	for _, v := range g.Nodes {
		switch v.(type) {
		case *Network:
			ret = append(ret, v.(*Network))
		}
	}

	sort.Sort(networks(ret))

	return ret
}

// return a list of nodes that contain the string v in the key k
func (g *Graph) FindNodes(k string, v string) []Node {
	k = strings.ToLower(k)
	var ret []Node
	for _, n := range g.Nodes {
		if n.Match(k, v) {
			ret = append(ret, n)
		}
	}

	return ret
}

func (g *Graph) FindEndpoints(k string, v string) []*Endpoint {
	k = strings.ToLower(k)
	var ret []*Endpoint
	for _, n := range g.GetEndpoints() {
		if n.Match(k, v) {
			ret = append(ret, n)
		}
	}

	sort.Sort(endpoints(ret))

	return ret
}

func (g *Graph) FindNetworks(k string, v string) []*Network {
	k = strings.ToLower(k)
	var ret []*Network
	for _, n := range g.GetNetworks() {
		if n.Match(k, v) {
			ret = append(ret, n)
		}
	}
	return ret
}

// HasNode returns true if a given node is in the graph
func (g *Graph) HasNode(n Node) bool {
	_, ok := g.Nodes[n.ID()]
	return ok
}

func (g *Graph) Update(n Node) (Node, error) {
	if !g.HasNode(n) {
		return nil, fmt.Errorf("no such node %v", n)
	}

	g.Nodes[n.ID()] = n
	return n, nil
}

// connect an endpoint to a network on a given edge. A node may only have a
// single connection to a given network. The edge must belong to e.
func (g *Graph) Connect(e, n Node, edge *Edge) error {
	if !g.HasNode(e) {
		return fmt.Errorf("node %v not in graph", e)
	}
	if !g.HasNode(n) {
		return fmt.Errorf("node %v not in graph", n)
	}

	if e.Type() != TYPE_ENDPOINT {
		return fmt.Errorf("node %v not an endpoint", e)
	}
	if n.Type() != TYPE_NETWORK {
		return fmt.Errorf("node %v not a network", n)
	}

	var endpoint *Endpoint
	var network *Network
	endpoint = e.(*Endpoint)
	network = n.(*Network)

	if !endpoint.HasEdge(edge) {
		return fmt.Errorf("edge %v not in endpoint %v", edge, endpoint)
	}
	if endpoint.Connected(network.ID()) {
		return fmt.Errorf("endpoint %v already connected to net %v", endpoint, network)
	}

	edge.N = network.ID()
	network.Endpoints = append(network.Endpoints, endpoint.ID())

	return nil
}

// disconnect a node from a network.
func (g *Graph) Disconnect(n1, n2 Node) error {
	if !g.HasNode(n1) {
		return fmt.Errorf("node %v not in graph", n1)
	}
	if !g.HasNode(n2) {
		return fmt.Errorf("node %v not in graph", n2)
	}
	if !n1.Connected(n2.ID()) {
		return fmt.Errorf("node %v not connected to node %v", n1, n2)
	}

	var endpoint *Endpoint
	var network *Network
	if n1.Type() == TYPE_ENDPOINT {
		endpoint = n1.(*Endpoint)
	} else {
		network = n1.(*Network)
	}
	if n2.Type() == TYPE_ENDPOINT {
		endpoint = n2.(*Endpoint)
	} else {
		network = n2.(*Network)
	}
	if endpoint == nil || network == nil {
		return fmt.Errorf("node type mismatch: %v, %v", n1.Type(), n2.Type())
	}

	for _, v := range endpoint.Edges {
		if v.N == network.ID() {
			v.N = UNCONNECTED
			break
		}
	}
	for i, v := range network.Endpoints {
		if v == endpoint.ID() {
			network.Endpoints = append(network.Endpoints[:i], network.Endpoints[i+1:]...)
			break
		}
	}
	return nil
}
