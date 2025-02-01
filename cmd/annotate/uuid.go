// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"io/ioutil"

	"github.com/sandia-minimega/discovery/v2/pkg/commands"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

type CommandUUID struct {
	commands.Base // embed
}

func init() {
	base := commands.Base{
		Usage: "uuid",
		Short: "annotate with random UUIDs",
		Long: `
Annotate nodes with random UUIDs.
`,
	}
	commands.Append(&CommandUUID{base})
}

func (c *CommandUUID) Run() error {
	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		return err
	}

	for _, v := range endpoints {
		if _, ok := v.D["uuid"]; ok && !*f_overwrite {
			continue
		}

		v.D["uuid"] = generateUUID()
		log.Debug("updating node %v with uuid %v", v, v.D["uuid"])

		if err := UpdateEndpoint(v); err != nil {
			return err
		}
	}

	return nil
}

// generateUUID using /proc.
//
// TODO: convert to using rand? Is there a reason to use kernel UUID?
func generateUUID() string {
	log.Debugln("generateUUID")
	uuid, err := ioutil.ReadFile("/proc/sys/kernel/random/uuid")
	if err != nil {
		log.Error("generateUUID: %v", err)
		return "00000000-0000-0000-0000-000000000000"
	}
	uuid = uuid[:len(uuid)-1]
	log.Debug("generated UUID: %v", string(uuid))
	return string(uuid)
}
