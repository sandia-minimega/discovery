// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

// Based on https://github.com/esnet/oscars-topology-json/wiki/JSON-API

type Data struct {
	// Either "success" or "error". If "success" then it was able to grab the
	// topology. If "error" it was unable to grab and/or parse the topology.
	Status string
	// A human-readable description of an error.
	Message string
	// A list of objects that represent the topology of a domain. Each domain
	// contains connectivity information about the network it represents.
	Domains []Domain
	// A list of circuits and the paths they take.
	Circuits []Circuit
}

type Domain struct {
	// A URN that identifies the domain (e.g. urn:ogf:network:domain=es.net).
	ID string
	// This list of "nodes", generally equivalent to routers or switches, in a
	// given domain
	Nodes []Node
}

type Node struct {
	// A URN that identifies the node (e.g.
	// urn:ogf:network:domain=es.net:node=albu-cr1).
	ID string
	// An IP address or hostname that may represent a loopback of management
	// address of the device represented by the node
	Address string
	// A descriptive name of the device represented by the node
	Name string
	// The DNS hostName of a device represented by the node
	Hostname string
	// The angular distance of node north or south of the earth's equator
	// expressed in degrees
	Latitude string
	// The angular distance of the node east or west of the meridian at
	// Greenwich, England, or west of the standard meridian of a celestial
	// object, expressed in degrees
	Longitude string
	// This list of "ports", generally equivalent to an interface (virtual or
	// physical) on a router or switch
	Ports []Port
}

type Port struct {
	// A URN that identifies the port (e.g.
	// urn:ogf:network:domain=es.net:node=albu-cr1:port=ge-2/0/1).
	ID string
	// Bandwidth of the port in bps
	Capacity string // supposed to be int...
	// The maximum amount of bandwidth that can be reserved by circuits in bps
	MaximumReservableCapacity string // supposed to be int...
	// The minimum amount of bandwidth that can be reserved by a circuit in bps
	MinimumReservableCapacity int
	// The increments that bandwidth can be reserved by a circuit in bps
	Granularity string // supposed to be int...
	// The name of the interface this port represents
	Name string `json:"ifName"`
	// The description of the interface this port represents
	Description string `json:"ifDescription"`
	// The IP address of the interface this port represents
	IPAddress string
	// The netmask of the interface this port represents
	Netmask string
	// If this port is a virtual port or a sub-port, this is the ID of the
	// underlying physical port
	Over string
	// A list of objects that describe how this port connects to another port
	Links []Link
}

type Link struct {
	// A URN that identifies the link (e.g.
	// urn:ogf:network:domain=es.net:node=albu-cr1:port=ge-2/0/1:link=*).
	ID string
	// An IP or hostname that represents a link
	Name string
	// Indicates whether this is a "logical" name.
	NameType string
	// Explicitly says "unidirectional" if its not bidirectional. It's null
	// otherwise.
	Type string
	// The URN indicating to what this link connects. It should be an ID of
	// another link. If it's urn:ogf:network:domain=:node=:port=:link= then it
	// doesn't know what's on the other end.
	RemoteLinkID string
	// A value used by pathfinding application to determine the "cost" of using
	// a link. All else being equal, the lower the cost value then the more
	// desirable a link is to a path computation algorithm trying to find the
	// lowest cost path.
	TrafficEngineeringMetric string // supposed to be int...
	// Bandwidth of the link in bps
	Capacity int
	// The maximum amount of bandwidth that can be reserved by circuits in bps
	MaximumReservableCapacity int
	// The minimum amount of bandwidth that can be reserved by a circuit in bps
	MinimumReservableCapacity int
	// The increments that bandwidth can be reserved by a circuit in bps
	Granularity int
	// ls2c for ethernet links, tdm for sdh/sonet links, and psc-4 for mpls
	// links. Values are taken from GMPLS.
	SwitchingcapType string
	// Ethernet for ethernet, sdh for SDH/SONET, and packet for MPLS and IP
	// links. Taken from GMPLS.
	EncodingType string
	// The VLANs circuits are allowed to reserve.
	VlanRangeAvailability string
	// The MTU of the link
	InterfaceMTU string
	// Indicates if link is capable of translating VLANs.
	VlanTranslation bool
}

type Circuit struct {
	// A URN that is the globally unique identifier of the circuit. This is
	// what perfSOANR theoretically uses to identify the circuit.
	ID string
	// unix timestamp of circuit start time
	Start int
	// unix timestamp of circuit end time
	End int
	// Another name for the circuit. In ESnet it corresponds to the ID OSCARS
	// uses to identify the circuit at the web service layer.
	Name string
	// A user-provided description of the circuit's purpose
	Description string
	// Circuits are composed of segments where a segment is the path across
	// each domain, This is the ID each domain uses to identify its local
	// segment of the circuit. There will be two for each domain, one for the
	// forward direction and another for the reverse direction,
	SegmentIDs []string `json:"segment_ids"`
	// These are the details of each segment in the circuit.
	Segments []Segment
}

type Segment struct {
	// Identifers of the sement. It should map to one of the ids in the parent
	// circuit objects segment_ids list
	ID string
	// An ordered list of the ports this segment uses (i.e. the first port is
	// the ingress and the last is the egress, everything in the middle is the
	// route data travels to get between them). The strings are URNS and should
	// map to an id field of a object you will find in one of the domain
	// objects.
	Ports []string
}
