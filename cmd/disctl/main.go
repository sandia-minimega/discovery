// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"fmt"

	"github.com/sandia-minimega/discovery/v2/pkg/discovery"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

var (
	f_panic  = flag.Bool("panic", false, "panic on quit, producing stack traces for debugging")
	f_server = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
	dc       *discovery.Client
)

// server operations
var (
	f_daemonSave   = flag.String("save", "", "save current model to file")
	f_daemonLoad   = flag.String("load", "", "load a model from a file")
	f_newEndpoint  = flag.Bool("ne", false, "create a new endpoint")
	f_newNetwork   = flag.Bool("nn", false, "create a new network")
	f_connect      = flag.Bool("c", false, "connect two nodes")
	f_disconnect   = flag.Bool("d", false, "disconnect two nodes")
	f_remove       = flag.Bool("r", false, "remove a node")
	f_f            = flag.Bool("f", false, "find nodes")
	f_fn           = flag.Bool("fn", false, "find networks")
	f_update       = flag.Bool("u", false, "update fields of an endpoint")
	f_updateConfig = flag.Bool("update-config", false, "update or add a config field")
	f_deleteConfig = flag.Bool("delete-config", false, "delete a config field")
	f_getConfig    = flag.Bool("config", false, "list one or all config fields")
)

func main() {
	var resp string
	var err error

	flag.Parse()

	log.Init()

	// don't allow concurrent operations
	err = flagCheck()
	if err != nil {
		log.Fatalln(err)
	}

	dc = discovery.New(*f_server)

	defer func() {
		if err != nil {
			log.Errorln(err)
		} else if resp != "" {
			fmt.Println(resp)
		}
	}()

	if *f_daemonSave != "" {
		resp, err = daemonSave(*f_daemonSave)
		return
	}
	if *f_daemonLoad != "" {
		resp, err = daemonLoad(*f_daemonLoad)
		return
	}
	if *f_newEndpoint {
		resp, err = endpointInsert()
		return
	}
	if *f_newNetwork {
		resp, err = networkInsert()
		return
	}
	if *f_connect {
		resp, err = connect()
		return
	}
	if *f_disconnect {
		resp, err = disconnect()
		return
	}
	if *f_remove {
		resp, err = remove()
		return
	}
	if *f_f {
		resp, err = find()
		return
	}
	if *f_fn {
		resp, err = findNetworks()
		return
	}
	if *f_update {
		resp, err = update()
		return
	}
	if *f_updateConfig {
		resp, err = updateConfig()
		return
	}
	if *f_deleteConfig {
		resp, err = deleteConfig()
		return
	}
	if *f_getConfig {
		resp, err = getConfig()
		return
	}
}

func flagCheck() error {
	count := 0

	if *f_daemonSave != "" {
		count++
	}
	if *f_daemonLoad != "" {
		count++
	}
	if *f_newEndpoint {
		count++
	}
	if *f_newNetwork {
		count++
	}
	if *f_connect {
		count++
	}
	if *f_disconnect {
		count++
	}
	if *f_remove {
		count++
	}
	if *f_f {
		count++
	}
	if *f_fn {
		count++
	}
	if *f_update {
		count++
	}
	if *f_updateConfig {
		count++
	}
	if *f_deleteConfig {
		count++
	}
	if *f_getConfig {
		count++
	}

	switch count {
	case 0:
		return fmt.Errorf("not enough arguments")
	case 1:
		return nil
	default:
		return fmt.Errorf("cannot perform concurrent operations")
	}
}
