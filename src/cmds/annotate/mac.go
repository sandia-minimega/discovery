// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"fmt"
	"math/rand"

	"pkg/commands"
	log "pkg/minilog"

	"github.com/google/gopacket/macs"
)

var validMACPrefix [][3]byte

type CommandMAC struct {
	commands.Base // embed
}

func init() {
	base := commands.Base{
		Usage: "mac",
		Short: "annotate with random MACs",
		Long: `
Annotates nodes with random MAC addresses for edges with an IP but no MAC. All
MACs will be assigned from a pool of valid MAC vendors.
`,
	}
	commands.Append(&CommandMAC{base})
}

func (c *CommandMAC) Run() error {
	// assume we are only run once...
	for k, _ := range macs.ValidMACPrefixMap {
		validMACPrefix = append(validMACPrefix, k)
	}

	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		return err
	}

	for _, v := range endpoints {
		for _, edg := range v.Edges {
			// skip edges without and IP
			if _, ok := edg.D["ip"]; !ok {
				continue
			}

			if _, ok := edg.D["mac"]; ok && !*f_overwrite {
				continue
			}

			edg.D["mac"] = randomMAC(rng)
			log.Debug("updating node %v with mac %v", v, edg.D["mac"])

			if err := UpdateEndpoint(v); err != nil {
				return err
			}
		}
	}

	return nil
}

// generate a random mac address and return as a string
func randomMAC(rng *rand.Rand) string {
	// pick a random valid prefix
	prefix := validMACPrefix[rng.Intn(len(validMACPrefix))]

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", prefix[0], prefix[1], prefix[2], rng.Intn(256), rng.Intn(256), rng.Intn(256))
}
