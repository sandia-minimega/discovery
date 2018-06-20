// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"fmt"
	"os"

	"pkg/discovery"
	log "pkg/minilog"
)

var (
	f_type   = flag.String("type", "cisco", "specify config type: [cisco, arista, brocade, juniper]")
	f_server = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
	f_dryrun = flag.Bool("dry-run", false, "do a dry run and do not push data to the server")
)

func usage() {
	fmt.Printf("USAGE: %v [OPTIONS] CONFIG\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Parse()

	log.Init()

	dc := discovery.New(*f_server)

	if flag.NArg() != 1 {
		usage()
	}

	var parser func(*os.File, *discovery.Client) error

	switch *f_type {
	case "cisco":
		parser = parseCisco
	case "brocade":
		parser = parseBrocade
	case "arista":
		parser = parseArista
	case "juniper":
		parser = parseJuniper
	default:
		log.Error("invalid config type")
		usage()
	}

	filename := flag.Arg(0)
	log.Debug("using filename: %v", filename)

	f, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}

	if err := parser(f, dc); err != nil {
		log.Fatalln(err)
	}
}
