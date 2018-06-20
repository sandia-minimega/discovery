// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bytes"
	"net"

	"github.com/google/gopacket/layers"
)

func IsEthernetBroadcast(mac net.HardwareAddr) bool {
	return bytes.Equal(mac, layers.EthernetBroadcast)
}

func (s *State) HandleIP() {
	// TODO: Anything to emit here?
}
