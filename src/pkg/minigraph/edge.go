// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package minigraph

import (
	"fmt"
	"strings"
)

// An endpoint network. Edges contain a reference to a connected network, or
// UNCONNECTED.
type Edge struct {
	N int
	D map[string]string
}

func newEdge() *Edge {
	return &Edge{
		D: make(map[string]string),
	}
}

func (e *Edge) Match(k, v string) bool {
	if k != "" {
		if k == "n" {
			if fmt.Sprintf("%v", e.N) == v {
				return true
			}
		}

		if val, ok := e.D[k]; ok {
			if strings.Contains(val, v) {
				return true
			}
		}
	} else {
		if fmt.Sprintf("%v", e.N) == v {
			return true
		}
		for _, val := range e.D {
			if strings.Contains(val, v) {
				return true
			}
		}
	}

	return false
}
