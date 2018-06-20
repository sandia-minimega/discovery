// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.
//
// p0f - TCP/IP packet matching
// ----------------------------
//
// Copyright (C) 2012 by Michal Zalewski <lcamtuf@coredump.cx>
//
// Distributed under the terms and conditions of GNU LGPL.

package p0f

import (
	"encoding/binary"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const (
	MinTCP4 = 40 // Minimum size of IPv4 header + TCP header
	MinTCP6 = 60 // Minimum size of IPv6 header + TCP header
)

// Different ways that the WSize field can be interpreted
const (
	WSizeTypeAny int = iota
	WSizeTypeNormal
	WSizeTypeMod
	WSizeTypeMSS
	WSizeTypeMTU
)

// For parsing list of options from the fingerprint file
var TCPOpts = map[string]uint8{
	"nop":  layers.TCPOptionKindNop,
	"mss":  layers.TCPOptionKindMSS,
	"ws":   layers.TCPOptionKindWindowScale,
	"sok":  layers.TCPOptionKindSACKPermitted,
	"sack": layers.TCPOptionKindSACK,
	"ts":   layers.TCPOptionKindTimestamps,
}

const (
	TCPQuirkECN        int = 1 << iota // ECN supported
	TCPQuirkDF                         // DF used (probably PMTUD); ignored for IPv6
	TCPQuirkNZID                       // Non-zero IDs when DF set; ignored for IPv6
	TCPQuirkZeroID                     // Zero IDs when DF not set; ignored for IPv6
	TCPQuirkNZMBZ                      // IP "must be zero" field isn't; ignored for IPv6
	TCPQuirkFlow                       // IPv6 flows used; ignored for IPv4
	TCPQuirkZeroSEQ                    // SEQ is zero
	TCPQuirkNZACK                      // ACK non-zero when ACK flag not set
	TCPQuirkZeroACK                    // ACK is zero when ACK flag set
	TCPQuirkNZURG                      // URG non-zero when URG flag not set
	TCPQuirkURG                        // URG flag set
	TCPQuirkPUSH                       // PUSH flag on a control packet
	TCPQuirkOptZeroTS1                 // Own timestamp set to zero
	TCPQuirkOptNZTS2                   // Peer timestamp non-zero on SYN
	TCPQuirkOptEOLNZ                   // Non-zero padding past EOL
	TCPQuirkOptEXWS                    // Excessive window scaling
	TCPQuirkOptBAD                     // Problem parsing TCP options
)

// For parsing list of quirks from the fingerprint file
var TCPQuirks = map[string]int{
	"df":     TCPQuirkDF,
	"id+":    TCPQuirkNZID,
	"id-":    TCPQuirkZeroID,
	"ecn":    TCPQuirkECN,
	"0+":     TCPQuirkNZMBZ,
	"flow":   TCPQuirkFlow,
	"seq-":   TCPQuirkZeroSEQ,
	"ack+":   TCPQuirkNZACK,
	"ack-":   TCPQuirkZeroACK,
	"uptr+":  TCPQuirkNZURG,
	"urgf+":  TCPQuirkURG,
	"pushf+": TCPQuirkPUSH,
	"ts1-":   TCPQuirkOptZeroTS1,
	"ts2+":   TCPQuirkOptNZTS2,
	"opt+":   TCPQuirkOptEOLNZ,
	"exws":   TCPQuirkOptEXWS,
	"bad":    TCPQuirkOptBAD,
}

// Parsed representation of a TCP fingerprint. See ParseTCPSignature.
type TCPSignature struct {
	Label string // type:class:name:flavor
	Raw   string // raw signature that this was parsed from

	Version      *int    // IPv4 or IPv6, (nil => any)
	ITTL         uint8   // initial TTL
	OptLen       uint8   // length of IPv4 options or IPv6 extension headers
	MSS          *uint16 // maximum segment size, (nil => any)
	WSizeType    int     // tells how to use the WSize field
	WSize        uint16  // window size
	WScale       *uint8  // window scaling factor, (nil => any)
	OptLayout    []uint8 // ordering of TCP options, if any
	Quirks       int     // quirks in IP or TCP headers
	PayloadClass int     // payload size classification

	badTTL bool
	EOLPad int // number of bytes after EOL to 32 byte padding

	parseError error
}

// TCPSyn stores information required for matching a TCP SYN or SYN+ACK against
// a TCPSignature. Built from State using NewTCPSyn.
type TCPSyn struct {
	HeaderLen uint16

	Quirks       int
	MSS          uint16
	WScale       uint8
	TS1, TS2     uint32
	PayloadClass int
}

type Packet interface {
	IP() gopacket.Layer

	TCP() *layers.TCP
}

// Compute the TCPSyn summary info from State.
func NewTCPSyn(p Packet) TCPSyn {
	syn := TCPSyn{}

	// Compute the IP/TCP header size
	syn.HeaderLen += uint16(p.TCP().DataOffset) * 4
	switch ip := p.IP().(type) {
	case *layers.IPv4:
		syn.HeaderLen += ip.Length - uint16(len(ip.Payload))
	case *layers.IPv6:
		syn.HeaderLen += ip.Length - uint16(len(ip.Payload))
	default:
		// that's strange
	}

	// Parse the TCP flags and extract needed values/quirks.
	for _, opt := range p.TCP().Options {
		switch opt.OptionType {
		case layers.TCPOptionKindMSS:
			if opt.OptionLength != 4 {
				syn.Quirks |= TCPQuirkOptBAD
			} else {
				syn.MSS = binary.BigEndian.Uint16(opt.OptionData[:2])
			}
		case layers.TCPOptionKindWindowScale:
			if opt.OptionLength != 3 {
				syn.Quirks |= TCPQuirkOptBAD
			} else {
				syn.WScale = uint8(opt.OptionData[0])
				// Maximum window scale is 14 according to RFC 1323
				if syn.WScale > 14 {
					syn.Quirks |= TCPQuirkOptEXWS
				}
			}
		case layers.TCPOptionKindSACKPermitted:
			if opt.OptionLength != 2 {
				syn.Quirks |= TCPQuirkOptBAD
			}
		case layers.TCPOptionKindTimestamps:
			if opt.OptionLength != 10 {
				syn.Quirks |= TCPQuirkOptBAD
			} else {
				syn.TS1 = binary.BigEndian.Uint32(opt.OptionData[:4])
				if syn.TS1 == 0 {
					syn.Quirks |= TCPQuirkOptZeroTS1
				}

				// Odd that the client sets TS2 when it hasn't connected yet
				syn.TS2 = binary.BigEndian.Uint32(opt.OptionData[4:8])
				if !p.TCP().ACK && syn.TS2 != 0 {
					syn.Quirks |= TCPQuirkOptNZTS2
				}
			}
		}
	}

	// Look for non-zero bytes in the padding
	for _, b := range p.TCP().Padding {
		if b != 0 {
			syn.Quirks |= TCPQuirkOptEOLNZ
			break
		}
	}

	// Look for internet-layer quirks
	switch ip := p.IP().(type) {
	case *layers.IPv4:
		// Lower two bits set => congestion control
		if ip.TOS&0x3 != 0 {
			syn.Quirks |= TCPQuirkECN
		}

		if ip.Flags&layers.IPv4MoreFragments != 0 {
			syn.Quirks |= TCPQuirkNZMBZ
		}

		if ip.Flags&layers.IPv4DontFragment != 0 {
			syn.Quirks |= TCPQuirkDF

			if ip.Id != 0 {
				syn.Quirks |= TCPQuirkNZID
			}
		} else if ip.Id == 0 {
			syn.Quirks |= TCPQuirkZeroID
		}
	case *layers.IPv6:
		if ip.FlowLabel != 0 {
			syn.Quirks |= TCPQuirkFlow
		}

		// Lower two bits set => congestion control
		if ip.TrafficClass&0x3 != 0 {
			syn.Quirks |= TCPQuirkECN
		}
	default:
		// that's strange
	}

	// Look for TCP-layer quirks
	if p.TCP().ECE || p.TCP().CWR || p.TCP().NS {
		syn.Quirks |= TCPQuirkECN
	}
	if p.TCP().Seq == 0 {
		syn.Quirks |= TCPQuirkZeroSEQ
	}
	if p.TCP().ACK {
		if p.TCP().Ack == 0 {
			syn.Quirks |= TCPQuirkZeroACK
		}
	} else if p.TCP().Ack != 0 && !p.TCP().RST {
		syn.Quirks |= TCPQuirkNZACK
	}
	if p.TCP().URG {
		syn.Quirks |= TCPQuirkURG
	} else if p.TCP().Urgent != 0 {
		syn.Quirks |= TCPQuirkNZURG
	}
	if p.TCP().PSH {
		syn.Quirks |= TCPQuirkPUSH
	}

	// Detect the payload class
	if len(p.TCP().Payload) > 0 {
		syn.PayloadClass = 1
	}

	return syn
}

// Match the TCP packet against sig. Matches might be fuzzy, if the quirks
// don't match exactly. This returned through the fuzzy parameter.
func (sig *TCPSignature) Match(p Packet, fuzzy *bool) bool {
	*fuzzy = false

	var ipv6 bool

	switch ip := p.IP().(type) {
	case *layers.IPv4:
		if !sig.match4(ip) {
			return false
		}
	case *layers.IPv6:
		ipv6 = true
		if !sig.match6(ip) {
			return false
		}
	default:
		// that's strange
	}

	// Simple checks
	if len(p.TCP().Padding) != sig.EOLPad {
		return false
	}
	if len(p.TCP().Options) != len(sig.OptLayout) {
		return false
	}

	syn := NewTCPSyn(p)

	// Check wildcards
	if sig.MSS != nil && syn.MSS != *sig.MSS {
		return false
	}
	if sig.WScale != nil && syn.WScale != *sig.WScale {
		return false
	}
	if sig.PayloadClass != -1 && syn.PayloadClass != sig.PayloadClass {
		return false
	}

	// Check window size matches
	switch sig.WSizeType {
	case WSizeTypeNormal:
		if p.TCP().Window != sig.WSize {
			return false
		}
	case WSizeTypeMod:
		if p.TCP().Window%sig.WSize != 0 {
			return false
		}
	case WSizeTypeMSS:
		win := p.TCP().Window

		if sig.WSize != syn.detectWinMultMSS(win, ipv6) {
			return false
		}
	case WSizeTypeMTU:
		win := p.TCP().Window

		if sig.WSize != syn.detectWinMultMTU(win, ipv6) {
			return false
		}
	}

	// Adjust quirks based on incoming packet version if signature applies to
	// both IPv4 and IPv6 packets
	quirks := sig.Quirks
	if sig.Version == nil {
		switch p.IP().LayerType() {
		case layers.LayerTypeIPv4:
			quirks &= ^(TCPQuirkFlow)
		case layers.LayerTypeIPv6:
			quirks &= ^(TCPQuirkDF | TCPQuirkNZID | TCPQuirkZeroID | TCPQuirkNZMBZ)
		}
	}

	if syn.Quirks != quirks {
		deleted := (quirks ^ syn.Quirks) & quirks
		added := (quirks ^ syn.Quirks) & syn.Quirks

		// If the deleted quirks are not just "df" or "id+"
		if deleted&^(TCPQuirkDF|TCPQuirkNZID) != 0 {
			return false
		}
		// If the added  quirks are not just "id-" or "ecn"
		if added&^(TCPQuirkZeroID|TCPQuirkECN) != 0 {
			return false
		}

		*fuzzy = true
	}

	// All checks passed!
	return true
}

// match4 checks the filter against all IPv4-specific fields.
func (sig *TCPSignature) match4(ip *layers.IPv4) bool {
	// Check for the correct version
	if sig.Version != nil && *sig.Version != 4 {
		return false
	}

	// Check that TTL doesn't exceed initial
	if ip.TTL > sig.ITTL {
		return false
	}

	// Check the option length, optLen *should* be less that 256
	optLen := ip.Length - 20
	optLen -= uint16(len(ip.Payload))
	if uint8(optLen) != sig.OptLen {
		return false
	}

	return true
}

// match6 checks the filter against all IPv6-specific fields.
func (sig *TCPSignature) match6(ip *layers.IPv6) bool {
	// Check for the correct version
	if sig.Version != nil && *sig.Version != 6 {
		return false
	}

	// Check that TTL doesn't exceed initial
	if ip.HopLimit > sig.ITTL {
		return false
	}

	return true
}

// TCP Window Size should be a multiple of the TCP MSS. There are a
// bunch of variations so try to find one that is a cleanly divides it.
func (syn *TCPSyn) detectWinMultMSS(win uint16, ip6 bool) uint16 {
	if syn.MSS < 100 {
		return 0
	}

	if win%syn.MSS == 0 {
		return win / syn.MSS
	}

	if syn.TS1 != 0 && win%(syn.MSS-12) == 0 {
		return win / (syn.MSS - 12)
	}

	if win%(1500-MinTCP4) == 0 {
		return win / (1500 - MinTCP4)
	}

	if win%(1500-MinTCP4-12) == 0 {
		return win / (1500 - MinTCP4 - 12)
	}

	if ip6 {
		if win%(1500-MinTCP6) == 0 {
			return win / (1500 - MinTCP6)
		}

		if win%(1500-MinTCP6-12) == 0 {
			return win / (1500 - MinTCP6 - 12)
		}
	}

	return 0
}

// TCP Window Size should be a multiple of the IP MTU. There are a bunch of
// variations so try to find one that is a cleanly divides it.
func (syn *TCPSyn) detectWinMultMTU(win uint16, ip6 bool) uint16 {
	if syn.MSS < 100 {
		return 0
	}

	if win%(syn.MSS+MinTCP4) == 0 {
		return win / (syn.MSS + MinTCP4)
	}

	if ip6 && win%(syn.MSS+MinTCP4) == 0 {
		return win / (syn.MSS + MinTCP4)
	}

	if win%(syn.MSS+syn.HeaderLen) == 0 {
		return win / (syn.MSS + syn.HeaderLen)
	}

	if win%1500 == 0 {
		return win / 1500
	}

	return 0
}
