// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Base implements the boring parts of the Command interface.
type Base struct {
	Usage       string
	Short, Long string

	Flags *flag.FlagSet
}

func (c *Base) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *Base) PrintUsage() {
	fmt.Printf("USAGE: %v [OPTION]... %v\n", os.Args[0], c.Usage)
	fmt.Println()

	fmt.Println(strings.TrimSpace(c.Long))
	fmt.Println()

	fmt.Printf("Universal options:\n")
	flag.PrintDefaults()

	if c.Flags != nil {
		fmt.Println()
		fmt.Printf("Command options:\n")
		c.Flags.PrintDefaults()
	}
}

func (c *Base) Name() string {
	return strings.Split(c.Usage, " ")[0]
}

// Listing returns command name and short help separated by tab
func (c *Base) Listing() string {
	return fmt.Sprintf("%v\t%v\n", c.Name(), c.Short)
}
