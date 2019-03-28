// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
	"syscall"
)

var (
	f_panic = flag.Bool("panic", false, "panic on quit, producing stack traces for debugging")
	f_file  = flag.String("f", "", "filename of graph to use/create")
	f_serve = flag.String("serve", fmt.Sprintf(":%v", discovery.Port), "web service address")
	f_web   = flag.String("web", "misc/web/", "path to static web content")
)

func main() {
	var err error

	flag.Parse()

	log.Init()

	if *f_file != "" {
		err = daemonLoad(*f_file)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		graph = minigraph.New()
		config = make(map[string]string)
	}

	// start the web service
	log.Debugln("starting web service")
	go web()

	// set up signal handling
	sig := make(chan os.Signal, 1024)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	log.Debugln("caught signal")

	if *f_panic {
		panic("stacktrace")
	}
	os.Exit(0)
}
