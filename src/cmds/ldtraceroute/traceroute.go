package main

import (
	"bufio"
	"os"
	"strings"
)

type hop struct {
	Hostname string
	IP       string
}

func parseTraceRoute(f *os.File) []*hop {
	var hops []*hop
	scanner := bufio.NewScanner(f)

	// skip first line
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Fields(line)
		if len(fields) < 6 {
			// Bail if there's a malformed record such as a router that doesn't
			// respond to traceroute. Hopefully we'll get at least a few hops
			// before we fail...
			break
		}

		var hostname string
		ip := strings.Trim(fields[2], "()")
		if fields[1] != ip {
			hostname = fields[1]
		}

		// TODO: record latency
		hops = append(hops, &hop{
			Hostname: hostname,
			IP:       ip,
		})
	}
	return hops
}
