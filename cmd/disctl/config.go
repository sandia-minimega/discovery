// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"flag"
	"fmt"
)

func updateConfig() (string, error) {
	args := flag.Args()
	if len(args) != 2 {
		return "", fmt.Errorf("invalid arguments: %v", args)
	}

	key := args[0]
	value := args[1]

	err := dc.SetConfig(key, value)
	return "", err
}

func deleteConfig() (string, error) {
	args := flag.Args()
	if len(args) != 1 {
		return "", fmt.Errorf("invalid arguments: %v", args)
	}

	key := args[0]

	err := dc.DeleteConfig(key)
	return "", err
}

func getConfig() (string, error) {
	args := flag.Args()
	if len(args) != 1 && len(args) != 0 {
		return "", fmt.Errorf("invalid arguments: %v", args)
	}

	config, err := dc.GetConfig()
	if err != nil {
		return "", err
	}

	if len(args) == 0 {
		// TODO: pretty print
		return fmt.Sprintf("%v", config), nil
	} else {
		key := args[0]
		if v, ok := config[key]; !ok {
			return "", fmt.Errorf("no such key: %v", key)
		} else {
			return v, nil
		}
	}
}
