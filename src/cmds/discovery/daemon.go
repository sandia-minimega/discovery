// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

type store struct {
	Config map[string]string
	Graph  []byte
}

func init() {
	gob.Register(&store{})
}

func webDaemon(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	// urls can be:
	//	/daemon/save/<path>
	//	/daemon/load/<path>

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	p := strings.Split(r.URL.Path, "/")[2:]
	log.Debug("split path: %v", p)

	switch r.Method {
	case "GET":
		if len(p) < 2 {
			err := fmt.Errorf("not enough arguments: %v", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		path := filepath.Join(p[1:]...)
		log.Debug("using path: %v", path)

		switch strings.ToLower(p[0]) {
		case "save":
			err := daemonSave(path)
			if err != nil {
				log.Errorln(err)
				w.WriteHeader(http.StatusInternalServerError)
				discovery.WriteError(w, r, err)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "load":
			err := daemonLoad(path)
			if err != nil {
				log.Errorln(err)
				w.WriteHeader(http.StatusInternalServerError)
				discovery.WriteError(w, r, err)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			err := fmt.Errorf("invalid daemon operation: %v", p[0])
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

}

func daemonLoad(path string) error {
	// read/create the specified file
	log.Debug("daemon load: %v", path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	s := &store{}
	dec := gob.NewDecoder(f)
	err = dec.Decode(s)
	if err != nil {
		return err
	}

	config = s.Config
	b := bytes.NewBuffer(s.Graph)

	// create the graph from the input file
	graph, err = minigraph.Read(b)
	return err
}

func daemonSave(path string) error {
	log.Debug("daemon save: %v", path)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var b bytes.Buffer

	err = graph.Write(&b)
	if err != nil {
		return err
	}

	s := &store{
		Config: config,
		Graph:  b.Bytes(),
	}

	return gob.NewEncoder(f).Encode(s)
}
