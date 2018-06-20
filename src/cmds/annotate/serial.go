// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"fmt"

	"pkg/commands"
	log "pkg/minilog"
)

type CommandSerial struct {
	commands.Base // embed
}

func init() {
	base := commands.Base{
		Usage: "serial",
		Short: "annotate with random serial numbers",
		Long: `
Annotate nodes with random serial numbers. Serial numbers are 8 hex characters.
`,
	}

	commands.Append(&CommandSerial{base})
}

func (r *CommandSerial) Run() error {
	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		return err
	}

	for _, v := range endpoints {
		if _, ok := v.D["serial"]; ok && !*f_overwrite {
			continue
		}

		// generate 8-character wide hex serial number
		v.D["serial"] = fmt.Sprintf("%08x", rng.Int())
		log.Debug("updating node %v with serial %v", v, v.D["serial"])

		if err := UpdateEndpoint(v); err != nil {
			return err
		}
	}

	return nil
}
