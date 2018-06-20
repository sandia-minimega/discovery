// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"pkg/discovery"
	log "pkg/minilog"
)

var (
	config map[string]string
)

// webConfig handles listing, adding, deleting, and modifying config parameters
// and supports the following methods:
//	GET
//		/config			list all config values
//		/config/<field>		list a config by a field
//	POST
//		/config/<field>		add/update a field
//	DELETE
//		/config/<field>		delete an config
//
func webConfig(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	p := strings.Split(r.URL.Path, "/")[1:]
	log.Debug("split path: %v", p)

	switch r.Method {
	case "GET":
		if len(p) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		// return all configs or key search
		var b []byte
		var err error
		if strings.TrimSpace(p[1]) == "" {
			b, err = json.MarshalIndent(config, "", "    ")
		} else {
			b, err = json.MarshalIndent(config[p[1]], "", "    ")
		}
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	case "POST":
		if len(p) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		k := strings.TrimSpace(p[1])
		if k == "" {
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		var data bytes.Buffer
		io.Copy(&data, r.Body)

		config[k] = data.String()

		w.WriteHeader(http.StatusCreated)
	case "DELETE":
		switch len(p) {
		case 2: // delete by key
			if strings.TrimSpace(p[1]) == "" {
				w.WriteHeader(http.StatusBadRequest)
				discovery.WriteError(w, r, fmt.Errorf("delete requires a key"))
				return
			} else {
				if _, ok := config[p[1]]; ok {
					delete(config, p[1])
				}
				w.WriteHeader(http.StatusOK)
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
