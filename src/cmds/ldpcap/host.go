// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"fmt"
	"io"
	"net"
	"strings"

	log "pkg/minilog"
)

type Host struct {
	// We may detect multiple operating systems for a single host so we store
	// the OS and it's weight from every event.
	OS map[OS]uint

	Nameservers        map[string]bool
	Routers            map[string]bool
	Hostnames          map[Hostname]bool
	Services           map[Service]bool
	AdvertisedServices map[string]bool

	IPs  map[string]net.IP
	MACs map[string]net.HardwareAddr

	// If we haven't seen any DHCP traffic for the host, we track it by its IP
	// address as an `external` host. If, in the future, we do see DHCP traffic
	// for this IP, we will forget about the `external` host and relearn the
	// host attributes.
	External bool

	// Set to true if the Host is a router
	Router bool
}

// HostOut represents Host for serialization
type HostOut struct {
	OS map[string]float64 `json:"os,omitempty"`

	Nameservers        []string   `json:"nameservers,omitempty"`
	Routers            []string   `json:"routers,omitempty"`
	Hostnames          []Hostname `json:"hostnames,omitempty"`
	Services           []Service  `json:"services,omitempty"`
	AdvertisedServices []string   `json:"advertised_services,omitempty"`

	IPs  []string `json:"ips,omitempty"`
	MACs []string `json:"macs,omitempty"`

	External bool `json:"external"`
	Router   bool `json:"router"`
}

func NewHost() *Host {
	return &Host{
		OS:                 map[OS]uint{},
		Nameservers:        map[string]bool{},
		Routers:            map[string]bool{},
		Hostnames:          map[Hostname]bool{},
		Services:           map[Service]bool{},
		AdvertisedServices: map[string]bool{},
		IPs:                map[string]net.IP{},
		MACs:               map[string]net.HardwareAddr{},
	}
}

// Merge two hosts. You should probably forget about other afterwards...
func (h *Host) Merge(other *Host) {
	log.Debug("Merging %v and %v", h.IPs, other.IPs)

	for k, v := range other.IPs {
		h.IPs[k] = v
	}
	for k, v := range other.MACs {
		h.MACs[k] = v
	}
	for k, v := range other.OS {
		h.OS[k] += v
	}
	for k := range other.Services {
		h.Services[k] = true
	}
	for k := range other.AdvertisedServices {
		h.AdvertisedServices[k] = true
	}
	for k := range other.Hostnames {
		h.Hostnames[k] = true
	}
	for k := range other.Nameservers {
		h.Nameservers[k] = true
	}
	for k := range other.Routers {
		h.Routers[k] = true
	}
}

func (h *Host) Write(out io.Writer) {
	fmt.Fprintf(out, "external=%t\n", h.External)

	fmt.Fprintf(out, "router=%t\n", h.Router)

	for ip := range h.IPs {
		fmt.Fprintf(out, "ip=%v\n", ip)
	}

	for mac := range h.MACs {
		fmt.Fprintf(out, "mac=%v\n", mac)
	}

	if v := fmtOS(h.OS); v != "" {
		fmt.Fprintf(out, "os=%v\n", v)
	}

	if v := fmtServices(h.Services); v != "" {
		fmt.Fprintf(out, "services=%v\n", v)
	}

	if v := fmtStrings(h.AdvertisedServices); v != "" {
		fmt.Fprintf(out, "advertised-services=%v\n", v)
	}

	if v := fmtHostnames(h.Hostnames); v != "" {
		fmt.Fprintf(out, "hostnames=%v\n", v)
	}

	if v := fmtStrings(h.Nameservers); v != "" {
		fmt.Fprintf(out, "nameservers=%v\n", v)
	}

	if v := fmtStrings(h.Routers); v != "" {
		fmt.Fprintf(out, "routers=%v\n", v)
	}

	fmt.Fprintln(out, "\n")
}

func (h *Host) Out() *HostOut {
	out := &HostOut{
		OS:       calcOS(h.OS),
		External: h.External,
		Router:   h.Router,
	}

	for v := range h.Nameservers {
		out.Nameservers = append(out.Nameservers, v)
	}

	for v := range h.Routers {
		out.Routers = append(out.Routers, v)
	}

	for v := range h.Hostnames {
		out.Hostnames = append(out.Hostnames, v)
	}

	for v := range h.Services {
		out.Services = append(out.Services, v)
	}

	for v := range h.AdvertisedServices {
		out.AdvertisedServices = append(out.AdvertisedServices, v)
	}

	for _, v := range h.IPs {
		out.IPs = append(out.IPs, v.String())
	}

	for v := range h.MACs {
		out.MACs = append(out.MACs, v)
	}

	return out
}

func calcOS(vals map[OS]uint) map[string]float64 {
	var sum float64
	res := map[string]float64{}

	// Compute OS -> total weight
	for os, weight := range vals {
		val := float64(weight)
		// Cut the weight of fuzzy results
		if os.Fuzzy {
			val /= 2
		}
		res[os.Label] += val
		sum += val
	}

	// Normalize
	for os := range res {
		res[os] /= sum
	}

	return res
}

func fmtOS(vals map[OS]uint) string {
	if len(vals) == 0 {
		return ""
	}

	return fmt.Sprintf("%v", calcOS(vals))
}

func fmtServices(vals map[Service]bool) string {
	keys := []string{}
	for k := range vals {
		keys = append(keys, fmt.Sprintf("%v", k))
	}

	return strings.Join(keys, ",")
}

func fmtHostnames(vals map[Hostname]bool) string {
	keys := []string{}
	for k := range vals {
		keys = append(keys, fmt.Sprintf("%q", k))
	}

	return strings.Join(keys, ",")
}

func fmtStrings(vals map[string]bool) string {
	keys := []string{}
	for k := range vals {
		keys = append(keys, fmt.Sprintf("%q", k))
	}

	return strings.Join(keys, ",")
}
