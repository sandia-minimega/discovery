// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

func (s *State) HandleTCP() {
	// Perform OS detection on SYN packets using p0f fingerprints
	if s.tcp.SYN {
		var fuzzy bool

		fmatch, match := -1, -1
		for i, sig := range TCPSigs {
			if sig.Match(s, &fuzzy) {
				if fuzzy {
					fmatch = i
				} else {
					match = i
					break
				}
			}
		}

		if match != -1 {
			s.events <- &EventOS{
				IP: s.SrcIP(),
				OS: OS{
					Label: TCPSigs[match].Label,
				},
			}
		} else if fmatch != -1 {
			s.events <- &EventOS{
				IP: s.SrcIP(),
				OS: OS{
					Label: TCPSigs[fmatch].Label,
					Fuzzy: true,
				},
			}
		}
	}

	// Use SYN-ACK packets to detect newly established connections (assuming
	// that no one is hand-crafting packets).
	if s.tcp.SYN && s.tcp.ACK {
		// SYN-ACK is sent from the server => client
		s.events <- &EventService{
			IP: s.SrcIP(),
			Service: Service{
				Internet:  s.internet,
				Transport: s.transport,
				Port:      uint16(s.tcp.SrcPort),
			},
		}
	}
}
