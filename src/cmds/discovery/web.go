// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

// rest (rest-like?) interface
// 	/ 		graph visualization
// 	/node		all nodes
// 	/node/<x>	node(s) by search field
// 	/endpoint	all endpoints
// 	/endpoint/<x>	endpoint(s) by search field
// 	/network	all networks
// 	/network/<x>	network(s) by search field
//	/walk

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

func init() {
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(*f_web))))
	http.HandleFunc("/nodes/", muer(webNodes))
	http.HandleFunc("/endpoints/", muer(webEndpoints))
	http.HandleFunc("/networks/", muer(webNetworks))
	http.HandleFunc("/neighbors/", muer(webNeighbors))
	http.HandleFunc("/walk/", muer(webWalk))
	http.HandleFunc("/daemon/", muer(webDaemon))
	http.HandleFunc("/connect/", muer(webConnect))
	http.HandleFunc("/disconnect/", muer(webDisconnect))
	http.HandleFunc("/config/", muer(webConfig))
	http.HandleFunc("/image/", muer(webImage))
}

var mu sync.Mutex

func muer(fn func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		fn(w, r)
	}
}

func web() {
	log.Debug("starting web service on %v", *f_serve)
	log.Fatalln(http.ListenAndServe(*f_serve, nil))
}

func webImage(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	p := strings.Split(r.URL.Path, "/")[2:]
	log.Debug("split path: %v", p)

	switch r.Method {
	case "GET":
		if len(p) != 1 {
			err := fmt.Errorf("not enough arguments: %v", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		log.Debug("using NID: %v", p[0])

		endpoint := graph.FindEndpoints("nid", p[0])
		if endpoint == nil {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			err := fmt.Errorf("no such endpoint: %v", p[0])
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		if d, ok := endpoint[0].D["image"]; !ok {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			err := fmt.Errorf("endpoint has no image!")
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		} else {
			// return the base64 decoded image
			img, err := base64.StdEncoding.DecodeString(d)
			if err != nil {
				w.Header().Set("Content-Type", "application/json; charset=UTF-8")
				w.WriteHeader(http.StatusBadRequest)
				discovery.WriteError(w, r, err)
				return
			}
			w.Write(img)
			return
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// connect an endpoint to a network. Nodes must be specified by NID. The URL
// must be of the form:
//	/connect/<network nid>/<endpoint nid>
//	/connect/<network nid>/<endpoint nid>/<edge index>
//
// If no edge index is specified, a new one is created.
func webConnect(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	p := strings.Split(r.URL.Path, "/")[2:]
	log.Debug("split path: %v", p)

	switch r.Method {
	case "POST":
		if len(p) < 2 {
			err := fmt.Errorf("not enough arguments: %v", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		log.Debug("using NIDs: %v, %v", p[0], p[1])

		endpoint := graph.FindEndpoints("nid", p[1])
		if endpoint == nil {
			err := fmt.Errorf("no such endpoint: %v", p[1])
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		network := graph.FindNetworks("nid", p[0])
		if network == nil {
			err := fmt.Errorf("no such network: %v", p[0])
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		var edge *minigraph.Edge

		if len(p) == 3 {
			log.Debug("using edge index: %v", p[2])
			eid, err := strconv.Atoi(p[2])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				discovery.WriteError(w, r, err)
				return
			}

			if len(endpoint[0].Edges) <= eid {
				err := fmt.Errorf("invalid edge id: %v", p[2])
				w.WriteHeader(http.StatusBadRequest)
				discovery.WriteError(w, r, err)
				return
			}

			edge = endpoint[0].Edges[eid]
		} else {
			// a new edge
			edge = endpoint[0].NewEdge()
		}

		err := graph.Connect(endpoint[0], network[0], edge)
		if err != nil {
			endpoint[0].Edges = endpoint[0].Edges[:len(endpoint[0].Edges)-1]
			log.Errorln(err)
			w.WriteHeader(http.StatusInternalServerError)
			discovery.WriteError(w, r, err)
			return
		}

		n := graph.FindEndpoints("nid", p[1])
		b, err := json.MarshalIndent(n[0], "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// disconnect an endpoint to a network. Nodes must be specified by NID. The URL
// must be of the form:
//	/connect/<network nid>/<endpoint nid>
func webDisconnect(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	p := strings.Split(r.URL.Path, "/")[2:]
	log.Debug("split path: %v", p)

	switch r.Method {
	case "POST":
		if len(p) < 2 {
			err := fmt.Errorf("not enough arguments: %v", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		log.Debug("using NIDs: %v, %v", p[0], p[1])

		endpoint := graph.FindEndpoints("nid", p[1])
		if endpoint == nil {
			err := fmt.Errorf("no such endpoint: %v", p[1])
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		network := graph.FindNetworks("nid", p[0])
		if network == nil {
			err := fmt.Errorf("no such network: %v", p[0])
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		err := graph.Disconnect(endpoint[0], network[0])
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(http.StatusInternalServerError)
			discovery.WriteError(w, r, err)
			return
		}

		n := graph.FindEndpoints("nid", p[1])
		b, err := json.MarshalIndent(n[0], "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// webNode handles listing, adding, deleting, and modifying nodes and supports
// the following methods:
//	GET
//		/nodes			list all nodes
//		/nodes/<field>/<value>	find nodes by a field
//		/nodes/<value>
//	DELETE
//		/nodes/<field>/<value>	delete a node
//		/nodes/<value>
//
func webNodes(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	p := strings.Split(r.URL.Path, "/")[1:]
	log.Debug("split path: %v", p)

	switch r.Method {
	case "GET":
		var nodes []minigraph.Node
		switch len(p) {
		case 2: // return all nodes or freeform search
			if strings.TrimSpace(p[1]) == "" {
				nodes = graph.GetNodes()
			} else {
				nodes = graph.FindNodes("", p[1])
			}
		case 3: // search
			nodes = graph.FindNodes(p[1], p[2])
		default:
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		b, err := json.MarshalIndent(nodes, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	case "DELETE":
		var nodes []minigraph.Node
		switch len(p) {
		case 2: // delete by freeform search
			if strings.TrimSpace(p[1]) == "" {
				w.WriteHeader(http.StatusBadRequest)
				discovery.WriteError(w, r, fmt.Errorf("delete requires a search term"))
				return
			} else {
				nodes = graph.FindNodes("", p[1])
			}
		case 3: // search
			nodes = graph.FindNodes(p[1], p[2])
		default:
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		for _, v := range nodes {
			err := graph.Delete(v)
			if err != nil {
				log.Errorln(err)
				w.WriteHeader(http.StatusInternalServerError)
				discovery.WriteError(w, r, err)
				return
			}
		}

		// write out the nodes we deleted
		b, err := json.MarshalIndent(nodes, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// webEndpoints handles listing, adding, deleting, and modifying endpoints and
// supports the following methods:
//	GET
//		/endpoints			list all endpoints
//		/endpoints/<field>/<value>	find endpoints by a field
//		/endpoints/<value>
//	POST
//		/endpoints			insert a new endpoint
//	PUT
//		/endpoints			update an endpoint
//	DELETE
//		/endpoints/<field>/<value>	delete an endpoint
//		/endpoints/<value>
//
func webEndpoints(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	p := strings.Split(r.URL.Path, "/")[1:]
	log.Debug("split path: %v", p)

	switch r.Method {
	case "GET":
		var endpoints []*minigraph.Endpoint
		switch len(p) {
		case 2: // return all endpoints or freeform search
			if strings.TrimSpace(p[1]) == "" {
				endpoints = graph.GetEndpoints()
			} else {
				endpoints = graph.FindEndpoints("", p[1])
			}
		case 3: // search
			endpoints = graph.FindEndpoints(p[1], p[2])
		default:
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		b, err := json.MarshalIndent(endpoints, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	case "POST":
		var data bytes.Buffer
		io.Copy(&data, r.Body)

		// create an endpoint
		var endpoints []*minigraph.Endpoint
		err := json.Unmarshal(data.Bytes(), &endpoints)
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
			return
		}

		var nodes []minigraph.Node
		for _, v := range endpoints {
			if v.D == nil {
				v.D = make(map[string]string)
			}
			n, err := graph.Insert(v)
			if err != nil {
				log.Errorln(err)
				w.WriteHeader(http.StatusInternalServerError)
				discovery.WriteError(w, r, err)
				return
			}
			nodes = append(nodes, n)
		}

		b, err := json.MarshalIndent(nodes, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write(b)
		}
	case "PUT":
		var data bytes.Buffer
		io.Copy(&data, r.Body)

		// get the list of updated endpoints
		var endpoints []*minigraph.Endpoint
		err := json.Unmarshal(data.Bytes(), &endpoints)
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
			return
		}

		// update each one based on NID
		var nodes []minigraph.Node
		for _, v := range endpoints {
			if v.D == nil {
				v.D = make(map[string]string)
			}
			n, err := graph.Update(v)
			if err != nil {
				log.Errorln(err)
				w.WriteHeader(http.StatusInternalServerError)
				discovery.WriteError(w, r, err)
				return
			}
			nodes = append(nodes, n)
		}

		b, err := json.MarshalIndent(nodes, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write(b)
		}
	case "DELETE":
		var endpoints []*minigraph.Endpoint
		switch len(p) {
		case 2: // delete by freeform search
			if strings.TrimSpace(p[1]) == "" {
				w.WriteHeader(http.StatusBadRequest)
				discovery.WriteError(w, r, fmt.Errorf("delete requires a search term"))
				return
			} else {
				endpoints = graph.FindEndpoints("", p[1])
			}
		case 3: // search
			endpoints = graph.FindEndpoints(p[1], p[2])
		default:
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		for _, v := range endpoints {
			err := graph.Delete(v)
			if err != nil {
				log.Errorln(err)
				w.WriteHeader(http.StatusInternalServerError)
				discovery.WriteError(w, r, err)
				return
			}
		}

		// write out the endpoints we deleted
		b, err := json.MarshalIndent(endpoints, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// webNetworks handles listing, adding, deleting, and modifying networks and
// supports the following methods:
//	GET
//		/networks			list all networks
//		/networks/<field>/<value>	find networks by a field
//		/networks/<value>
//	POST
//		/networks			insert a new network
//	PUT
//		/networks			update an network
//	DELETE
//		/networks/<field>/<value>	delete an network
//		/networks/<value>
//
func webNetworks(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	p := strings.Split(r.URL.Path, "/")[1:]
	log.Debug("split path: %v", p)

	switch r.Method {
	case "GET":
		var networks []*minigraph.Network
		switch len(p) {
		case 2: // return all networks or freeform search
			if strings.TrimSpace(p[1]) == "" {
				networks = graph.GetNetworks()
			} else {
				networks = graph.FindNetworks("", p[1])
			}
		case 3: // search
			networks = graph.FindNetworks(p[1], p[2])
		default:
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		b, err := json.MarshalIndent(networks, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	case "POST":
		var data bytes.Buffer
		io.Copy(&data, r.Body)

		// create an network
		var networks []*minigraph.Network
		err := json.Unmarshal(data.Bytes(), &networks)
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
			return
		}

		var nodes []minigraph.Node
		for _, v := range networks {
			n, err := graph.Insert(v)
			if err != nil {
				log.Errorln(err)
				w.WriteHeader(http.StatusInternalServerError)
				discovery.WriteError(w, r, err)
				return
			}
			nodes = append(nodes, n)
		}

		b, err := json.MarshalIndent(nodes, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write(b)
		}
	case "PUT":
		var data bytes.Buffer
		io.Copy(&data, r.Body)

		// get the list of updated networks
		var networks []*minigraph.Network
		err := json.Unmarshal(data.Bytes(), &networks)
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
			return
		}

		// update each one based on NID
		var nodes []minigraph.Node
		for _, v := range networks {
			n, err := graph.Update(v)
			if err != nil {
				log.Errorln(err)
				w.WriteHeader(http.StatusInternalServerError)
				discovery.WriteError(w, r, err)
				return
			}
			nodes = append(nodes, n)
		}

		b, err := json.MarshalIndent(nodes, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write(b)
		}
	case "DELETE":
		var networks []*minigraph.Network
		switch len(p) {
		case 2: // delete by freeform search
			if strings.TrimSpace(p[1]) == "" {
				w.WriteHeader(http.StatusBadRequest)
				discovery.WriteError(w, r, fmt.Errorf("delete requires a search term"))
				return
			} else {
				networks = graph.FindNetworks("", p[1])
			}
		case 3: // search
			networks = graph.FindNetworks(p[1], p[2])
		default:
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		for _, v := range networks {
			err := graph.Delete(v)
			if err != nil {
				log.Errorln(err)
				w.WriteHeader(http.StatusInternalServerError)
				discovery.WriteError(w, r, err)
				return
			}
		}

		// write out the networks we deleted
		b, err := json.MarshalIndent(networks, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// webNeighbors handles listing neighbors to a given node and supports the
// following methods:
//	GET
//		/neighbors/<field>/<value>	find nodes by a field
//		/neighbors/<value>
//
func webNeighbors(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	p := strings.Split(r.URL.Path, "/")[1:]
	log.Debug("split path: %v", p)

	switch r.Method {
	case "GET":
		var nodes []minigraph.Node
		switch len(p) {
		case 2: // reeform search
			if strings.TrimSpace(p[1]) == "" {
				w.WriteHeader(http.StatusBadRequest)
				discovery.WriteError(w, r, fmt.Errorf("invalid search term"))
			} else {
				nodes = graph.FindNodes("", p[1])
				if len(nodes) > 1 {
					w.WriteHeader(http.StatusBadRequest)
					discovery.WriteError(w, r, fmt.Errorf("search term not unique"))
					return
				}
			}
		case 3: // search
			nodes = graph.FindNodes(p[1], p[2])
			if len(nodes) > 1 {
				discovery.WriteError(w, r, fmt.Errorf("search term not unique"))
				return
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
			// TODO: write Allow in the header
			return
		}

		nodeIDs := nodes[0].Neighbors()
		nodes = []minigraph.Node{}
		for _, v := range nodeIDs {
			nodes = append(nodes, graph.Nodes[v])
		}

		b, err := json.MarshalIndent(nodes, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func webWalk(w http.ResponseWriter, r *http.Request) {
	log.Info("%v\t%v", r.Method, r.RequestURI)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	p := strings.Split(r.URL.Path, "/")[1:]
	log.Debug("split path: %v", p)

	if len(p) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch strings.ToLower(p[1]) {
	case "nodes":
		// create a new node walker
		id := NewWalker(minigraph.TYPE_NODE)
		url := fmt.Sprintf("/walk/%v", id)
		http.Redirect(w, r, url, http.StatusFound)
		return
	case "endpoints":
		// create a new endpoint walker
		id := NewWalker(minigraph.TYPE_ENDPOINT)
		url := fmt.Sprintf("/walk/%v", id)
		http.Redirect(w, r, url, http.StatusFound)
		return
	case "networks":
		// create a new network walker
		id := NewWalker(minigraph.TYPE_NETWORK)
		url := fmt.Sprintf("/walk/%v", id)
		http.Redirect(w, r, url, http.StatusFound)
		return
	default:
		// hopefully a walker id
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		id, err := strconv.Atoi(p[1])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		n, err := WalkerNext(id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			discovery.WriteError(w, r, err)
			return
		}

		b, err := json.MarshalIndent(n, "", "    ")
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(422)
			discovery.WriteError(w, r, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	}
}
