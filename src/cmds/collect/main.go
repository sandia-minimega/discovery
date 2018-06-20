// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"fmt"

	"pkg/commands"
	"pkg/discovery"
	log "pkg/minilog"
)

// universal flags
var (
	f_server = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
	f_dryrun = flag.Bool("dry-run", false, "print updates rather than commit them")
)

var (
	dc *discovery.Client
)

func main() {
	flag.Parse()

	log.Init()

	dc = discovery.New(*f_server)

	if err := commands.Run(); err != nil {
		log.Errorln(err)
	}
}
