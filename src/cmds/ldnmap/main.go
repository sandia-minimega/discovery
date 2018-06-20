// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"pkg/discovery"
	"pkg/minigraph"
	log "pkg/minilog"
)

var (
	f_server = flag.String("server", fmt.Sprintf("localhost:%v", discovery.Port), "web service")
	dc       *discovery.Client
)

type Data struct {
	Hosts []host `xml:"host"`
}

type host struct {
	Addr   Address `xml:"address"`
	Ports  ports   `xml:"ports"`
	OS     nmapos  `xml:"os"`
	Status status  `xml:"status"`
}

type status struct {
	State string `xml:"state,attr"`
}

type nmapos struct {
	Match osmatch `xml:"osmatch"`
}

type osmatch struct {
	Name string `xml:"name,attr"`
}

type ports struct {
	Ports []port `xml:"port"`
}

type port struct {
	Protocol string `xml:"protocol,attr"`
	PortID   string `xml:"portid,attr"`
	State    state  `xml:"state"`
}

type state struct {
	State string `xml:"state,attr"`
}

type Address struct {
	IP   string `xml:"addr,attr"`
	Type string `xml:"addrtype,attr"`
}

func main() {
	flag.Parse()

	log.Init()

	dc = discovery.New(*f_server)

	args := flag.Args()
	if len(args) != 1 {
		log.Fatal("invalid arguments: %v", args)
	}
	filename := args[0]
	log.Debug("using filename: %v", filename)

	xmldata, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}

	data := &Data{}

	err = xml.Unmarshal(xmldata, data)
	if err != nil {
		log.Fatalln(err)
	}

	log.Debug("processed %v hosts", len(data.Hosts))

	for _, v := range data.Hosts {
		if v.Status.State != "up" {
			continue
		}

		// for now just create a new node, we'll merge another day
		e := &minigraph.Endpoint{}
		es, err := dc.InsertEndpoints(e)
		if err != nil {
			log.Fatalln(err)
		}
		e = es[0]

		// populate the endpoint
		if v.Addr.IP != "" {
			// figure out which network this belongs on
			newip, n := findNet(v.Addr.IP)
			if n == nil {
				// add a new network
				n = &minigraph.Network{}
				ns, err := dc.InsertNetworks(n)
				n = ns[0]
				if err != nil {
					log.Fatalln(err)
				}
				newip = v.Addr.IP
			}

			e, err = dc.Connect(n.ID(), e.ID(), discovery.EDGE_NONE)
			if err != nil {
				log.Fatalln(err)
			}
			edg := e.Edges[len(e.Edges)-1]
			edg.D["ip"] = newip
		}
		for _, p := range v.Ports.Ports {
			if p.State.State == "open" {
				if ports, ok := e.D["ports"]; ok {
					e.D["ports"] = fmt.Sprintf("%v,%v", ports, p.PortID)
				} else {
					e.D["ports"] = p.PortID
				}
			}
		}
		osm := v.OS.Match.Name
		if osm != "" {
			e.D["osmatch"] = osm
			if strings.Contains(osm, "Linux") {
				e.D["os"] = "linux"
			} else if strings.Contains(osm, "Windows") {
				e.D["os"] = "windows"
			}
		}

		_, err = dc.UpdateEndpoints(e)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func findNet(ip string) (string, *minigraph.Network) {
	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		log.Fatalln(err)
	}
	for _, e := range endpoints {
		for _, edg := range e.Edges {
			if dip, ok := edg.D["ip"]; ok {
				_, ipn, err := net.ParseCIDR(dip)
				if err != nil {
					log.Errorln(err)
					continue
				}
				if ipn.Contains(net.ParseIP(ip)) {
					newip := &net.IPNet{
						IP:   net.ParseIP(ip),
						Mask: ipn.Mask,
					}
					ns, err := dc.GetNetworks("nid", fmt.Sprintf("%v", edg.N))
					if err != nil {
						log.Fatalln(err)
					}
					n := ns[0]
					return newip.String(), n
				}
			}
		}
	}
	return "", nil
}
