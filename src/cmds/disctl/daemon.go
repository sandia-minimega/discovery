// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

func daemonSave(path string) (string, error) {
	return "", dc.Save(path)
}

func daemonLoad(path string) (string, error) {
	return "", dc.Load(path)
}
