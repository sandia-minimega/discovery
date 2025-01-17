// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package minigraph

import (
	"fmt"
)

type Network struct {
	NID       int
	Endpoints []int
	D         map[string]string
}

func (n *Network) Data() map[string]string {
	return n.D
}

func (n *Network) setID(id int) {
	n.NID = id
}

func (n *Network) Match(k, v string) bool {
	if k != "" {
		switch k {
		case "nid":
			if fmt.Sprintf("%v", n.NID) == v {
				return true
			}
		}
	} else {
		if fmt.Sprintf("%v", n.NID) == v {
			return true
		}
	}

	return false
}

func (n *Network) String() string {
	return fmt.Sprintf("%v", n.ID())
}

// Return the ID of the network
func (n *Network) ID() int {
	return n.NID
}

// Return the type. Used as a fast alternative to reflection when using the
// Node interface.
func (n *Network) Type() int {
	return TYPE_NETWORK
}

// Connected returns true if the network is connected to the node referenced
// by the given node ID.
func (n *Network) Connected(e int) bool {
	for _, v := range n.Endpoints {
		if v == e {
			return true
		}
	}
	return false
}

// Return IDs of connected neighbors
func (n *Network) Neighbors() []int {
	var ret []int
	for _, v := range n.Endpoints {
		if v != UNCONNECTED {
			ret = append(ret, v)
		}
	}
	return ret
}
