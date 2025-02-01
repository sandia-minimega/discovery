// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/sandia-minimega/discovery/v2/pkg/commands"
	"github.com/sandia-minimega/discovery/v2/pkg/discovery"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

// universal flags
var (
	f_server    = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
	f_seed      = flag.Int64("seed", 0, "seed for random number generator, 0 means use random seed")
	f_dryrun    = flag.Bool("dry-run", false, "print updates rather than commit them")
	f_overwrite = flag.Bool("overwrite", false, "overwrite values even if already set")
)

var (
	dc  *discovery.Client
	rng *rand.Rand
)

func main() {
	flag.Parse()

	log.Init()

	dc = discovery.New(*f_server)

	if *f_seed != 0 {
		rng = rand.New(rand.NewSource(*f_seed))
	} else {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	if err := commands.Run(); err != nil {
		log.Errorln(err)
	}
}
