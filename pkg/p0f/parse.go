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
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ParseTCPSignature parses the p0f TCP signature format:
// 		ver:ittl:olen:mss:wsize,scale:olayout:quirks:pclass
func ParseTCPSignature(label, s string) (*TCPSignature, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 8 {
		return nil, errors.New("expected 8 fields")
	}

	sig := TCPSignature{Label: label, Raw: s}

	parseTCPSigVersion(parts[0], &sig)
	parseTCPSigITTL(parts[1], &sig)
	parseTCPSigOptLen(parts[2], &sig)
	parseTCPSigMSS(parts[3], &sig)

	parts2 := strings.Split(parts[4], ",")
	if len(parts2) != 2 {
		return nil, errors.New("expected wsize,scale")
	}

	parseTCPSigWSize(parts2[0], &sig)
	parseTCPSigWScale(parts2[1], &sig)

	parseTCPSigOptLayout(parts[5], &sig)
	parseTCPSigQuirks(parts[6], &sig)
	parseTCPSigPayloadClass(parts[7], &sig)

	if sig.parseError != nil {
		return nil, sig.parseError
	}

	return &sig, nil
}

func parseTCPSigVersion(s string, sig *TCPSignature) {
	if sig.parseError != nil {
		return
	}

	switch s {
	case "4":
		v := 4
		sig.Version = &v
	case "6":
		v := 6
		sig.Version = &v
	case "*":
		// Do nothing, use zero value
	default:
		sig.parseError = errors.New("invalid ip version")
	}
}

func parseTCPSigITTL(s string, sig *TCPSignature) {
	if sig.parseError != nil {
		return
	}

	if strings.HasSuffix(s, "-") {
		sig.badTTL = true
		s = s[:len(s)-1]
	}

	if ittl, err := strconv.Atoi(s); err != nil {
		sig.parseError = fmt.Errorf("expected integer for ittl, not %v", s)
	} else if ittl < 1 || ittl > 255 {
		sig.parseError = errors.New("1 <= ittl <= 255")
	} else {
		sig.ITTL = uint8(ittl)
	}
}

func parseTCPSigOptLen(s string, sig *TCPSignature) {
	if sig.parseError != nil {
		return
	}

	if olen, err := strconv.Atoi(s); err != nil {
		sig.parseError = fmt.Errorf("expected integer for olen, not %v", s)
	} else if olen < 0 || olen > 255 {
		sig.parseError = errors.New("0 <= olen <= 255")
	} else {
		sig.OptLen = uint8(olen)
	}
}

func parseTCPSigMSS(s string, sig *TCPSignature) {
	if sig.parseError != nil {
		return
	}

	if s == "*" {
		// Match any
	} else if mss, err := strconv.Atoi(s); err != nil {
		sig.parseError = fmt.Errorf("expected integer for mss, not %v", s)
	} else if mss < 0 || mss > 65535 {
		sig.parseError = errors.New("0 <= mss <= 65535")
	} else {
		mss := uint16(mss)
		sig.MSS = &mss
	}
}

func parseTCPSigWSize(s string, sig *TCPSignature) {
	if sig.parseError != nil {
		return
	}

	var min, max int

	if s == "" || s == "*" {
		sig.WSizeType = WSizeTypeAny
		return
	} else if strings.HasPrefix(s, "%") {
		sig.WSizeType = WSizeTypeMod
		s = s[1:]
		min, max = 2, 65535
	} else if strings.HasPrefix(s, "mss*") {
		sig.WSizeType = WSizeTypeMSS
		s = s[4:]
		min, max = 1, 1000
	} else if strings.HasPrefix(s, "mtu*") {
		sig.WSizeType = WSizeTypeMTU
		s = s[4:]
		min, max = 1, 1000
	} else {
		sig.WSizeType = WSizeTypeNormal
		min, max = 0, 65535
	}

	if wsize, err := strconv.Atoi(s); err != nil {
		sig.parseError = fmt.Errorf("expected integer for wsize, not %v", s)
	} else if sig.WSizeType != WSizeTypeAny && (wsize < min || wsize > max) {
		sig.parseError = fmt.Errorf("%v <= wsize <= %v", min, max)
	} else {
		sig.WSize = uint16(wsize)
	}
}

func parseTCPSigWScale(s string, sig *TCPSignature) {
	if sig.parseError != nil {
		return
	}

	if s == "*" {
		// Match any
	} else if wscale, err := strconv.Atoi(s); err != nil {
		sig.parseError = fmt.Errorf("expected integer for wscale, not %v", s)
	} else if wscale < 0 || wscale > 255 {
		sig.parseError = errors.New("0 <= wscale <= 255")
	} else {
		wscale := uint8(wscale)
		sig.WScale = &wscale
	}
}

// eol+n  - explicit end of options, followed by n bytes of padding
// nop    - no-op option
// mss    - maximum segment size
// ws     - window scaling
// sok    - selective ACK permitted
// sack   - selective ACK (should not be seen)
// ts     - timestamp
// ?n     - unknown option ID n
func parseTCPSigOptLayout(opts string, sig *TCPSignature) {
	for _, s := range strings.Split(opts, ",") {
		// Check on every iteration in case we encounter a bad option
		if sig.parseError != nil {
			return
		}

		if opt, ok := TCPOpts[s]; ok {
			sig.OptLayout = append(sig.OptLayout, opt)
		} else if strings.HasPrefix(s, "eol+") {
			s = s[4:]

			if pad, err := strconv.Atoi(s); err != nil {
				sig.parseError = fmt.Errorf("expected integer for eol pad, not %v", s)
			} else if pad < 0 || pad > 255 {
				sig.parseError = errors.New("0 <= eol pad <= 255")
			} else {
				sig.EOLPad = pad
			}

			// eol should be the last option
			break
		} else if s[:1] == "?" {
			s = s[1:]

			if opt, err := strconv.Atoi(s); err != nil {
				sig.parseError = fmt.Errorf("expected integer for unknown opt, not %v", s)
			} else if opt < 0 || opt > 255 {
				sig.parseError = errors.New("0 <= unknown opt <= 255")
			} else {
				opt := uint8(opt)
				sig.OptLayout = append(sig.OptLayout, opt)
			}
		} else {
			sig.parseError = errors.New("malformed option layout")
		}
	}
}

// df     - "don't fragment" set (probably PMTUD); ignored for IPv6
// id+    - DF set but IPID non-zero; ignored for IPv6
// id-    - DF not set but IPID is zero; ignored for IPv6
// ecn    - explicit congestion notification support
// 0+     - "must be zero" field not zero; ignored for IPv6
// flow   - non-zero IPv6 flow ID; ignored for IPv4
//
// seq-   - sequence number is zero
// ack+   - ACK number is non-zero, but ACK flag not set
// ack-   - ACK number is zero, but ACK flag set
// uptr+  - URG pointer is non-zero, but URG flag not set
// urgf+  - URG flag used
// pushf+ - PUSH flag used
//
// ts1-   - own timestamp specified as zero
// ts2+   - non-zero peer timestamp on initial SYN
// opt+   - trailing non-zero data in options segment
// exws   - excessive window scaling factor (> 14)
// bad    - malformed TCP options
func parseTCPSigQuirks(quirks string, sig *TCPSignature) {
	if sig.parseError != nil {
		return
	}

	if quirks == "" {
		return
	}

	for _, s := range strings.Split(quirks, ",") {
		if quirk, ok := TCPQuirks[s]; ok {
			sig.Quirks |= quirk
		} else {
			sig.parseError = fmt.Errorf("unknown quirk: %v", s)
			return
		}
	}

	// TODO: Check for incompatible quirk/ip version
}

func parseTCPSigPayloadClass(s string, sig *TCPSignature) {
	if sig.parseError != nil {
		return
	}

	switch s {
	case "*":
		sig.PayloadClass = -1
	case "0":
		sig.PayloadClass = 0
	case "+":
		sig.PayloadClass = 1
	default:
		sig.parseError = errors.New("invalid payload class")
	}
}
