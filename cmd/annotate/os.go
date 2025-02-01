// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/sandia-minimega/discovery/v2/pkg/commands"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

type OS struct {
	Name string

	// Prob is percentage (0 <= Prob <= 1)
	Prob float64
}

type OSes struct {
	// list of OS
	oses []OS

	// computed ranges for RNG:
	// 	 if ranges[i-1] < rand.Float64() < ranges[i] => pick OS i
	ranges []float64
}

type CommandOS struct {
	commands.Base // embed
}

func init() {
	base := commands.Base{
		Flags: flag.NewFlagSet("os", flag.ExitOnError),
		Usage: "os <OS,WEIGHT>...",
		Short: "annotate with random oses",
		Long: `
Annotate nodes with OSes based on the distribution specified on the command
line. For example, to get 50% Windows and 50% Linux:

	os Linux:0.5 Windows:0.5
`,
	}

	commands.Append(&CommandOS{base})
}

func (o OSes) Rand(r *rand.Rand) string {
	v := r.Float64()

	for i, v2 := range o.ranges {
		if v < v2 {
			return o.oses[i].Name
		}
	}

	log.Error("no OSes?")
	return "???"
}

func (c *CommandOS) Run() error {
	oses, err := parseOSes(c.Flags.Args())
	if err != nil {
		return err
	}

	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		return err
	}

	for _, v := range endpoints {
		if _, ok := v.D["os"]; ok && !*f_overwrite {
			continue
		}

		v.D["os"] = oses.Rand(rng)
		log.Debug("updating node %v with os %v", v, v.D["os"])

		if err := UpdateEndpoint(v); err != nil {
			return err
		}
	}

	return nil
}

// parseOSes parses OS,WEIGHT args and returns a map with normalized values
func parseOSes(args []string) (*OSes, error) {
	oses := []OS{}

	for _, f := range args {
		parts := strings.SplitN(f, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected OS,WEIGHT not `%v`", f)
		}

		v, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("expected float not `%v`", parts[1])
		}

		name := parts[0]

		log.Info("adding OS %v with weight %v", name, v)

		oses = append(oses, OS{Name: name, Prob: v})
	}

	// normalize weights to create probabilities
	sum := 0.0
	for _, v := range oses {
		sum += v.Prob
	}

	prev := 0.0
	ranges := []float64{}
	for i := range oses {
		oses[i].Prob = oses[i].Prob / sum

		log.Debug("converted OS %v weight to probability %v%%", oses[i].Name, oses[i].Prob*100)

		prev = prev + oses[i].Prob
		ranges = append(ranges, prev)
	}

	return &OSes{oses: oses, ranges: ranges}, nil
}
