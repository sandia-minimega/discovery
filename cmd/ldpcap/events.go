// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"encoding/binary"
	"hash/fnv"
	"io"
	"net"

	"github.com/google/gopacket/layers"
)

// TODO: We lose contextual information about events when we don't aggregate
// them per-packet or per-flow. Perhaps we need to add connection IDs to the
// events so that the inference engine can have more to work with.
// Alternatively, we could produce a slice of events for each packet and
// process them together.

// The stream deduper assigns weights to events based on how many duplicate
// events have been seen in the stream. This allows anomalous events to be
// `squashed` by being out weighed by frequent events. For example, if we see
// the same OS 10K times and then a different OS once, we will still say the OS
// we saw 10K times is more likely.

type Event interface {
	Hash() uint64
	SetWeight(uint)
}

type BaseEvent struct {
	Weight uint
}

type EventService struct {
	BaseEvent // embed
	Service   // embed

	IP net.IP
}

type EventAdvertisedService struct {
	BaseEvent // embed

	// List of service types: http://www.dns-sd.org/servicetypes.html
	Service  string
	Hostname string
	Port     uint16
}

type EventNameserver struct {
	BaseEvent // embed

	IP, Nameserver net.IP
}

type EventOS struct {
	BaseEvent // embed
	OS        // embed

	IP net.IP
}

type EventHostname struct {
	BaseEvent // embed
	Hostname  // embed

	IP net.IP
}

type EventDHCP struct {
	BaseEvent // embed

	MsgType layers.DHCPMsgType

	HardwareAddr net.HardwareAddr
	ClientIP     net.IP

	// From options
	RequestedIPAddr net.IP
	Subnet          net.IPNet

	Hostname    string
	Domain      string
	Nameservers []net.IP
	Routers     []net.IPNet
}

type EventEth struct {
	BaseEvent

	SrcMAC net.HardwareAddr
	Desc   string // vendor pulled from mac addr

}

type EventNeighbor struct {
	BaseEvent // embed

	HardwareAddr net.HardwareAddr
	IP           net.IP

	Router bool
}

type EventRouter struct {
	BaseEvent // embed

	HardwareAddr net.HardwareAddr
	IP           net.IP
	IPPrefixes   []net.IPNet
}

func (e *BaseEvent) SetWeight(weight uint) {
	e.Weight = weight
}

func (e EventService) Hash() uint64 {
	h := fnv.New64a()

	h.Write(e.IP)
	binary.Write(h, binary.LittleEndian, e.Port)

	return h.Sum64()
}

func (e EventAdvertisedService) Hash() uint64 {
	h := fnv.New64a()

	h.Write([]byte(e.Service))
	h.Write([]byte(e.Hostname))
	binary.Write(h, binary.LittleEndian, e.Port)

	return h.Sum64()
}

func (e EventNameserver) Hash() uint64 {
	h := fnv.New64a()

	h.Write(e.IP)
	h.Write(e.Nameserver)

	return h.Sum64()
}

func (e EventOS) Hash() uint64 {
	h := fnv.New64a()

	h.Write(e.IP)
	h.Write([]byte(e.Label))
	writeBool(h, e.Fuzzy)

	return h.Sum64()
}

func (e EventHostname) Hash() uint64 {
	h := fnv.New64a()

	h.Write(e.IP)
	h.Write([]byte(e.Name))
	binary.Write(h, binary.LittleEndian, e.Type)

	return h.Sum64()
}

func (e EventDHCP) Hash() uint64 {
	h := fnv.New64a()

	binary.Write(h, binary.LittleEndian, e.MsgType)
	h.Write(e.HardwareAddr)
	h.Write(e.ClientIP)

	h.Write(e.RequestedIPAddr)
	h.Write(e.Subnet.IP)
	h.Write(e.Subnet.Mask)

	h.Write([]byte(e.Hostname))
	h.Write([]byte(e.Domain))

	for _, ns := range e.Nameservers {
		h.Write(ns)
	}

	for _, v := range e.Routers {
		h.Write(v.IP)
		h.Write(v.Mask)
	}

	return h.Sum64()
}

func (e EventNeighbor) Hash() uint64 {
	h := fnv.New64a()

	h.Write(e.HardwareAddr)
	h.Write(e.IP)

	writeBool(h, e.Router)

	return h.Sum64()
}

func (e EventRouter) Hash() uint64 {
	h := fnv.New64a()

	h.Write(e.IP)
	h.Write(e.HardwareAddr)
	for _, ipp := range e.IPPrefixes {
		h.Write(ipp.IP)
		h.Write(ipp.Mask)
	}
	return h.Sum64()
}

func writeBool(w io.Writer, val bool) {
	if val {
		binary.Write(w, binary.LittleEndian, uint8(0x1))
	} else {
		binary.Write(w, binary.LittleEndian, uint8(0x0))
	}
}
