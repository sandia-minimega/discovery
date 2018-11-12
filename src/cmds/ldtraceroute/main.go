package main

import (
	"flag"
	"fmt"
	"os"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

var (
	f_server = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
)

type Client struct {
	*discovery.Client // embed
}

func usage() {
	fmt.Printf("USAGE: %v [OPTIONS] <traceroute file>\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	log.Init()

	c := Client{discovery.New(*f_server)}

	file, err := os.Open(flag.Args()[0])
	if err != nil {
		log.Fatalln(err)
	}
	tr := parseTraceRoute(file)

	var prev int

	// we don't know subnet so just create new networks along the hop path if a
	// given endpoint/network doesn't already exist
	for i, hop := range tr {
		endpoints, err := c.GetEndpoints("", hop.IP)
		if err != nil {
			log.Fatalln(err)
		}
		var e *minigraph.Endpoint
		if len(endpoints) == 0 {
			log.Debug("no endpoint found for %v, creating a new one", hop)
			e = &minigraph.Endpoint{
				D: make(map[string]string),
			}
			if hop.Hostname != "" {
				e.D["hostname"] = hop.Hostname
			}
			ret, err := c.InsertEndpoints(e)
			if err != nil {
				log.Fatalln(err)
			}
			e = ret[0]
		} else {
			e = endpoints[0]
		}
		log.Debugln(e)

		eidx := discovery.EDGE_NONE
		for j, v := range e.Edges {
			if v.Match("ip", hop.IP) {
				eidx = j
				break
			}
		}
		var n *minigraph.Network
		if eidx == discovery.EDGE_NONE {
			n = &minigraph.Network{}
			ret, err := c.InsertNetworks(n)
			if err != nil {
				log.Fatalln(err)
			}
			n = ret[0]
			e, err = c.Connect(n.NID, e.NID, eidx)
			if err != nil {
				log.Fatalln(err)
			}
			eidx = 0
			e.Edges[eidx].D["ip"] = hop.IP
			es, err := c.UpdateEndpoints(e)
			if err != nil {
				log.Fatalln(err)
			}
			e = es[0]
		} else {
			ns, err := c.GetNetworks("nid", fmt.Sprintf("%v", e.Edges[eidx].N))
			if err != nil {
				log.Fatalln(err)
			}
			n = ns[0]
		}

		// connect to the previous hop, which doesn't have an IP on
		// this side of the connection
		if i != 0 {
			ret, err := c.GetEndpoints("nid", fmt.Sprintf("%v", prev))
			if err != nil {
				log.Fatalln(err)
			}
			eLast := ret[0]

			connected := false
			// skip if we're already connected to the previous hop
			for _, x := range e.Neighbors() {
				nets, err := c.GetNetworks("nid", fmt.Sprintf("%v", x))
				if err != nil {
					log.Fatalln(err)
				}
				for _, y := range nets[0].Neighbors() {
					if y == prev {
						log.Debug("already connected: %v, %v", hop, tr[i-1])
						connected = true
						break
					}
				}
			}

			if !connected {
				_, err := c.Connect(n.NID, eLast.NID, discovery.EDGE_NONE)
				if err != nil {
					log.Fatalln(err)
				}
			}
		}

		prev = e.NID
	}
}
