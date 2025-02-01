// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/sandia-minimega/discovery/v2/pkg/minigraph"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

const (
	WALKER_RATE = time.Duration(30 * time.Second)
)

var (
	walkers    map[int]*walker
	walkerID   int
	walkerLock sync.Mutex
)

func init() {
	walkers = make(map[int]*walker)
	go walkerReaper()
}

type walker struct {
	id      int
	last    time.Time
	filter  int
	visited map[int]bool
}

func NewWalker(filter int) int {
	walkerLock.Lock()
	defer walkerLock.Unlock()

	walkerID++

	walkers[walkerID] = &walker{
		id:      walkerID,
		last:    time.Now(),
		filter:  filter,
		visited: make(map[int]bool),
	}

	return walkerID
}

func WalkerNext(wid int) (minigraph.Node, error) {
	walkerLock.Lock()
	defer walkerLock.Unlock()
	w, ok := walkers[wid]
	if !ok {
		return nil, fmt.Errorf("no such walker %v", wid)
	}

	n := w.next()
	if n == nil {
		delete(walkers, w.id)
	}

	return n, nil
}

func walkerReaper() {
	for {
		time.Sleep(WALKER_RATE)
		walkerLock.Lock()

		for k, v := range walkers {
			if time.Since(v.last) > WALKER_RATE {
				log.Debug("reaping expired walker %v", v.id)
				delete(walkers, k)
			}
		}

		walkerLock.Unlock()
	}
}

func (w *walker) next() minigraph.Node {
	// kick the dog
	w.last = time.Now()

	// the graph may have changed out from under us, so we do this the hard
	// way. There are plenty of optimizations for this later on if we need
	// to do so.
	// search the graph for an unvisited node
	for k, v := range graph.Nodes {
		if w.visited[k] {
			continue
		}
		w.visited[k] = true

		// visit only endpoints/networks if we're filtering
		if w.filter != minigraph.TYPE_NODE && w.filter != v.Type() {
			continue
		}

		return v
	}

	// we hit everything
	return nil
}
