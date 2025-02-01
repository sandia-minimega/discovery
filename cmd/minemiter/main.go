// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/sandia-minimega/discovery/v2/pkg/discovery"
	"github.com/sandia-minimega/discovery/v2/pkg/minigraph"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

var (
	f_templatePath = flag.String("path", "templates", "template path")
	f_server       = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
	f_output       = flag.String("w", "minemiter.mm", "output file")
	dc             *discovery.Client
)

const (
	MAX_TOKEN = 1024 * 1024
)

func main() {
	flag.Parse()

	log.Init()

	log.Debug("using path: %v", *f_templatePath)

	dc = discovery.New(*f_server)

	// prepare configuration parameters
	config, err := dc.GetConfig()
	if err != nil {
		log.Fatalln(err)
	}

	networks, err := dc.GetNetworks("", "")
	if err != nil {
		log.Fatalln(err)
	}

	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		log.Fatalln(err)
	}

	var nodes []minigraph.Node
	for _, n := range networks {
		nodes = append(nodes, n)
	}
	for _, e := range endpoints {
		nodes = append(nodes, e)
	}

	// get an ordered list of templates to apply and preprocess them
	err = parseTemplates(*f_templatePath)
	if err != nil {
		log.Fatalln(err)
	}

	// start parsing!
	output, err := parse(config, nodes)
	if err != nil {
		log.Fatalln(err)
	}

	err = ioutil.WriteFile(*f_output, output, 0664)
	if err != nil {
		log.Fatalln(err)
	}
}

func pretty(in bytes.Buffer) string {
	var out bytes.Buffer
	scanner := bufio.NewScanner(&in)
	var buf []byte
	scanner.Buffer(buf, MAX_TOKEN)

	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			_, err := out.WriteString(strings.TrimSpace(scanner.Text()) + "\n")
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln(err)
	}
	return out.String()
}
