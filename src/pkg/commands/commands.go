// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"
)

type Commands []Command

type Command interface {
	// Name of the command
	Name() string

	// PrintUsage block to os.Stdout
	PrintUsage()

	// Short listing consisting of name and short help separated by tab
	Listing() string

	// FlagSet for this command
	FlagSet() *flag.FlagSet

	// Callback
	Run() error
}

var DefaultCommands = Commands{}

func (c Commands) PrintUsage() {
	fmt.Printf("USAGE: %v [OPTION]... <COMMAND> [<args>]\n", os.Args[0])
	fmt.Println()

	fmt.Printf("Available commands:\n")
	names := []string{}
	for _, cmd := range c {
		names = append(names, cmd.Name())
	}
	sort.Strings(names)

	w := tabwriter.NewWriter(os.Stdout, 2, 0, 1, ' ', 0)
	for _, name := range names {
		cmd := c.Find(name)
		io.WriteString(w, "\t")
		io.WriteString(w, cmd.Listing())
	}
	w.Flush()
	fmt.Println()

	fmt.Printf("Universal options:\n")
	flag.PrintDefaults()
}

func (c Commands) Find(name string) Command {
	for _, cmd := range c {
		if name == cmd.Name() {
			return cmd
		}
	}

	return nil
}

// Run finds the correct subcommand and runs it.
func (c Commands) Run() error {
	if flag.NArg() == 0 {
		c.PrintUsage()
		return nil
	}

	// check for help
	if flag.Arg(0) == "help" && flag.NArg() == 1 {
		c.PrintUsage()
		return nil
	}
	if flag.Arg(0) == "help" && flag.NArg() == 2 {
		if cmd := c.Find(flag.Arg(1)); cmd != nil {
			cmd.PrintUsage()
			return nil
		}

		// must not have given a valid command
		return fmt.Errorf("invalid command, see `%v help`", os.Args[0])
	}

	// must be trying to run a command
	if cmd := c.Find(flag.Arg(1)); cmd != nil {
		if f := cmd.FlagSet(); f != nil {
			f.Parse(flag.Args()[1:])
		}

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%v failed: %v", err)
		}

		return nil
	}

	// must not have given a valid command
	return fmt.Errorf("invalid command, see `%v help`", os.Args[0])
}

func Append(c Command) {
	DefaultCommands = append(DefaultCommands, c)
}

func PrintUsage() {
	DefaultCommands.PrintUsage()
}

func Find(name string) Command {
	return DefaultCommands.Find(name)
}

func Run() error {
	return DefaultCommands.Run()
}
