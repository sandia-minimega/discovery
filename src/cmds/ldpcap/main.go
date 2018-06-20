// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bufio"
	"flag"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"time"

	log "pkg/minilog"
	"pkg/p0f"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var (
	f_hosts = flag.String("hosts", "", "hosts output filename")

	f_p0f = flag.String("p0f", "", "file containing p0f fingerprints")

	f_dot1q = flag.Bool("dot1q", false, "enable 802.1q (VLAN) analysis")
	f_icmp4 = flag.Bool("icmp4", false, "enable icmp4 analysis")
	f_dns   = flag.Bool("dns", false, "enable dns analysis")
	f_arp   = flag.Bool("arp", false, "enable arp analysis")
	f_dhcp  = flag.Bool("dhcp", false, "enable dhcp analysis")

	f_profile = flag.String("profile", "", "write cpu profile to file")

	f_push = flag.String("push", "", "read hosts output and push to specified server")
)

// used for live capture to signal when to stop
var CAUGHT_SIGNAL bool

var TCPSigs []*p0f.TCPSignature

func main() {
	flag.Parse()

	log.Init()

	if *f_push != "" {
		if *f_hosts == "" {
			log.Fatal("must specify host file when pushing to server")
		}

		pushHosts()
		return
	}

	if flag.NArg() == 0 {
		log.Fatal("must specify at least one input PCAP")
	}

	if *f_p0f != "" {
		if err := parseFingerprints(*f_p0f); err != nil {
			log.Fatal("unable to parse p0f fingerprints: %v", err)
		}
	}

	var hostsOut = os.Stdout

	if *f_hosts != "" {
		f, err := os.Create(*f_hosts)
		if err != nil {
			log.Fatal("unable to open hosts output file: %v", err)
		}
		hostsOut = f
	}

	if *f_profile != "" {
		log.Info("Enabling CPU profiling, writing to: %s", *f_profile)
		f, err := os.Create(*f_profile)
		if err != nil {
			log.Fatal(err.Error())
		}

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	state := State{
		events: make(chan Event),
	}

	state.DecodingLayerParser = gopacket.NewDecodingLayerParser(
		layers.LayerTypeEthernet,
		&state.eth,
		&state.ip4,
		&state.ip6,
	)

	if *f_dot1q {
		state.AddDecodingLayer(&state.dot1q)
	}
	if *f_icmp4 {
		state.AddDecodingLayer(&state.icmp4)
	}
	if *f_dns {
		state.AddDecodingLayer(&state.udp)
		state.AddDecodingLayer(&state.tcp)
		state.AddDecodingLayer(&state.dns)
	}
	if *f_dhcp {
		state.AddDecodingLayer(&state.udp)
		state.AddDecodingLayer(&state.dhcp)
	}

	if *f_arp {
		state.AddDecodingLayer(&state.arp)
	}

	go func() {
		defer close(state.events)

		for _, in := range flag.Args() {
			state.Run(in)
		}
	}()

	// graceful exit
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		<-c
		CAUGHT_SIGNAL = true
	}()

	inference := NewInference(dedupStream(state.events))

	inference.Run()
	inference.WriteHostsJSON(hostsOut)

	log.Info("%v", inference.stats)
}

func (s *State) SrcIP() net.IP {
	if s.internet == layers.LayerTypeIPv4 {
		return s.ip4.SrcIP
	} else if s.internet == layers.LayerTypeIPv6 {
		return s.ip6.SrcIP
	} else {
		return nil
	}
}

func (s *State) DstIP() net.IP {
	if s.internet == layers.LayerTypeIPv4 {
		return s.ip4.DstIP
	} else if s.internet == layers.LayerTypeIPv6 {
		return s.ip6.DstIP
	} else {
		return nil
	}
}

func (s *State) GuessDistance() uint8 {
	var ttl uint8

	if s.internet == layers.LayerTypeIPv4 {
		ttl = s.ip4.TTL
	} else if s.internet == layers.LayerTypeIPv6 {
		ttl = s.ip6.HopLimit
	}

	for i := uint8(0); i < 3; i++ {
		if max := uint8(32 << i); ttl <= max {
			return max - ttl
		}
	}
	return 255 - ttl
}

func (s *State) Run(in string) {
	log.Info("Processing %v", in)

	// Record how long it took us to process the file
	start := time.Now()
	defer func() {
		log.Info("Processed %v in %v", in, time.Now().Sub(start))
	}()

	packets, err := pcap.OpenOffline(in)
	if err != nil {
		// not a file, maybe an interface?
		packets, err = pcap.OpenLive(in, 1600, true, pcap.BlockForever)
		if err != nil {
			log.Fatal("Failed to open pcap %v -- %v", in, err)
		}
	}

	decodeFailed := map[string]int{}
	decodedLayers := []gopacket.LayerType{}

	for !CAUGHT_SIGNAL {
		data, ci, err := packets.ReadPacketData()
		if err != nil {
			if err != io.EOF {
				log.Error("Error reading packet data: ", err)
			}
			break
		}

		s.captureInfo = ci

		// for testing
		//	err = s.DecodeLayers(data, &decodedLayers)
		//	for _, typ := range decodedLayers {
		//		fmt.Println(" successfully decoded layer type", typ)
		//	}
		//	fmt.Println("err: ", err)
		//	fmt.Println("eth can decode: ", s.eth.CanDecode())

		if err := s.DecodeLayers(data, &decodedLayers); err != nil {
			switch err := err.(type) {
			case gopacket.UnsupportedLayerType:
				decodeFailed[gopacket.LayerType(err).String()] += 1
			default:
				log.Error("Error parsing packet: %v", err)
				continue
			}
		}

		// Process all the decoded layers
		for _, typ := range decodedLayers {
			s.HandleLayer(typ)
		}

		if s.Truncated {
			// TODO: Do we care? Probably could look for frequently truncated
			// packets from misbehaving hosts...

			//fmt.Println("  Packet has been truncated")
		}
	}

	if len(decodeFailed) > 0 {
		log.Info("Failed to decode:")
	}
	for layer, count := range decodeFailed {
		log.Info("  %v: %v", layer, count)
	}
}

func (s *State) HandleLayer(typ gopacket.LayerType) {
	switch typ {
	case layers.LayerTypeEthernet:
		s.link = typ
	case layers.LayerTypeDot1Q:
		s.HandleDot1Q()
	case layers.LayerTypeIPv4:
		s.internet = typ
		s.HandleIP()
	case layers.LayerTypeIPv6:
		s.internet = typ
		s.HandleIP()
	case layers.LayerTypeICMPv4:
		s.HandleICMPv4()
	case layers.LayerTypeARP:
		s.HandleARP()
	case layers.LayerTypeTCP:
		s.transport = layers.LayerTypeTCP
		s.HandleTCP()
	case layers.LayerTypeUDP:
		s.transport = layers.LayerTypeUDP
		s.HandleUDP()
	case layers.LayerTypeDNS:
		s.HandleDNS()
	case layers.LayerTypeDHCPv4:
		s.HandleDHCP()
	}
}

func dedupStream(in chan Event) chan Event {
	out := make(chan Event)

	type EventCounter struct {
		Hash  uint64
		Event Event
		Count uint
	}

	go func() {
		defer close(out)

		deduper := make([]EventCounter, 1<<20)

		for e := range in {
			hash := e.Hash()
			index := hash % uint64(len(deduper))

			counter := &deduper[index]

			// New event is not the same as the stored event
			if counter.Hash != hash {
				// Re-emit the old event, if there's a non-zero count
				if counter.Count > 0 {
					counter.Event.SetWeight(counter.Count)
					out <- counter.Event
				}

				// Emit and track new event
				e.SetWeight(1)
				out <- e
				counter.Hash = hash
				counter.Event = e
				counter.Count = 0
			} else {
				counter.Count += 1
			}
		}

		// Emit final set of events that have non-zero counts
		for _, counter := range deduper {
			if counter.Count > 0 {
				out <- counter.Event
			}
		}
	}()

	return out
}

func parseFingerprints(fname string) error {
	log.Debug("Parsing p0f fingerprints from %v", fname)

	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	label := ""
	interested := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip blank lines/comments
		if strings.HasPrefix(line, ";") {
			continue
		} else if len(line) == 0 {
			continue
		}

		// New section, only interested in the two tcp sections
		if strings.HasPrefix(line, "[") {
			interested = (line == "[tcp:request]" || line == "[tcp:response]")
		}

		if !interested {
			continue
		}

		if strings.HasPrefix(line, "label") {
			label = strings.TrimSpace(strings.Split(line, "=")[1])
		} else if strings.HasPrefix(line, "sig") {
			s := strings.TrimSpace(strings.Split(line, "=")[1])

			sig, err := p0f.ParseTCPSignature(label, s)
			if err != nil {
				return err
			}

			TCPSigs = append(TCPSigs, sig)
		}
	}

	log.Debug("Parsed %v TCP fingerprints", len(TCPSigs))

	return scanner.Err()
}
