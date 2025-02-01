// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package minigraph

import (
	"flag"
	"math/rand"
	"os"
	"testing"
	"time"

	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Verbose() {
		log.AddLogger("stdio", os.Stderr, log.DEBUG, true)
	}

	os.Exit(m.Run())
}

func TestSimple(t *testing.T) {
	// simple graph
	// e1 -- n1 -- e2

	g := New()

	e1 := g.NewEndpoint()
	e2 := g.NewEndpoint()
	n1 := g.NewNetwork()

	e1e := e1.NewEdge()
	e2e := e2.NewEdge()

	err := g.Connect(e1, n1, e1e)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Connect(e2, n1, e2e)
	if err != nil {
		t.Fatal(err)
	}

	e1n := e1.Neighbors()
	e2n := e2.Neighbors()
	n1n := n1.Neighbors()

	if len(e1n) != 1 || e1n[0] != n1.ID() {
		t.Fatalf("invalid edges on e1: %v", e1n)
	}
	if len(e2n) != 1 || e2n[0] != n1.ID() {
		t.Fatalf("invalid edges on e2: %v", e2n)
	}
	if len(n1n) != 2 || n1n[0] != e1.ID() || n1n[1] != e2.ID() {
		t.Fatalf("invalid edges on n1: %v", n1n)
	}
}

func TestDupEdge(t *testing.T) {
	g := New()

	e1 := g.NewEndpoint()
	n1 := g.NewNetwork()

	edge := e1.NewEdge()

	err := g.Connect(e1, n1, edge)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Connect(e1, n1, edge)
	if err == nil {
		t.Fatal("duplicate edge")
	}
}

func TestNodeDel(t *testing.T) {
	// e1 -- n1 -- e2
	// |---- n2 ---|

	g := New()

	e1 := g.NewEndpoint()
	e2 := g.NewEndpoint()
	n1 := g.NewNetwork()
	n2 := g.NewNetwork()

	e1e1 := e1.NewEdge()
	e1e2 := e1.NewEdge()
	e2e1 := e2.NewEdge()
	e2e2 := e2.NewEdge()

	err := g.Connect(e1, n1, e1e1)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Connect(e1, n2, e1e2)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Connect(e2, n1, e2e1)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Connect(e2, n2, e2e2)
	if err != nil {
		t.Fatal(err)
	}

	err = g.Delete(e2)
	if err != nil {
		t.Fatal(err)
	}

	// make sure it's sane
	if _, ok := g.Nodes[e2.ID()]; ok {
		t.Fatalf("node is still in graph")
	}

	// make sure the graph is correct
	// n1 -- s1
	// |---- s2

	e1e := e1.Neighbors()
	n1e := n1.Neighbors()
	n2e := n2.Neighbors()

	if len(e1e) != 2 || e1e[0] != n1.ID() || e1e[1] != n2.ID() {
		t.Fatalf("invalid edges on e1: %v", e1e)
	}
	if len(n1e) != 1 || n1e[0] != e1.ID() {
		t.Fatalf("invalid edges on n1: %v", n1e)
	}
	if len(n2e) != 1 || n2e[0] != e1.ID() {
		t.Fatalf("invalid edges on n2: %v", n2e)
	}
}

func TestDisconnect(t *testing.T) {
	// e1 -- n1 -- e2
	// |---- n2 ---|

	g := New()

	e1 := g.NewEndpoint()
	e2 := g.NewEndpoint()
	n1 := g.NewNetwork()
	n2 := g.NewNetwork()

	e1e1 := e1.NewEdge()
	e1e2 := e1.NewEdge()
	e2e1 := e2.NewEdge()
	e2e2 := e2.NewEdge()

	err := g.Connect(e1, n1, e1e1)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Connect(e1, n2, e1e2)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Connect(e2, n1, e2e1)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Connect(e2, n2, e2e2)
	if err != nil {
		t.Fatal(err)
	}

	err = g.Disconnect(n1, e2)

	// make sure the graph is correct
	// graph:
	// e1 -- n1    e2
	// |-----n2----|
	e1e := e1.Neighbors()
	e2e := e2.Neighbors()
	n1e := n1.Neighbors()
	n2e := n2.Neighbors()

	if len(e1e) != 2 || e1e[0] != n1.ID() || e1e[1] != n2.ID() {
		t.Fatalf("invalid edges on e1: %v", e1e)
	}
	if len(e2e) != 1 || e2e[0] != n2.ID() {
		t.Fatalf("invalid edges on e2: %v", e2e)
	}
	if len(n1e) != 1 || n1e[0] != e1.ID() {
		t.Fatalf("invalid edges on n1: %v", n1e)
	}
	if len(n2e) != 2 || n2e[0] != e1.ID() || n2e[1] != e2.ID() {
		t.Fatalf("invalid edges on n2: %v", n2e)
	}

}

func BenchmarkBigGraph(b *testing.B) {
	g := New()
	rand.Seed(time.Now().UnixNano())
	n := g.NewNetwork()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := g.NewEndpoint()
		edge := e.NewEdge()
		err := g.Connect(e, n, edge)
		if err != nil {
			b.Fatal(err)
		}
	}
}
