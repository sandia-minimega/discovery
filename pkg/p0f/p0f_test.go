// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package p0f

import (
	"testing"
)

var testSigs = []string{
	// s:unix:Linux:3.11 and newer
	`*:64:0:*:mss*20,10:mss,sok,ts,nop,ws:df,id+:0`,
	`*:64:0:*:mss*20,7:mss,sok,ts,nop,ws:df,id+:0`,

	// s:win:Windows:XP
	`*:128:0:*:16384,0:mss,nop,nop,sok:df,id+:0`,
	`*:128:0:*:65535,0:mss,nop,nop,sok:df,id+:0`,
	`*:128:0:*:65535,0:mss,nop,ws,nop,nop,sok:df,id+:0`,
	`*:128:0:*:65535,1:mss,nop,ws,nop,nop,sok:df,id+:0`,
	`*:128:0:*:65535,2:mss,nop,ws,nop,nop,sok:df,id+:0`,

	// s:unix:Mac OS X:10.x
	`*:64:0:*:65535,1:mss,nop,ws,nop,nop,ts,sok,eol+1:df,id+:0`,
	`*:64:0:*:65535,3:mss,nop,ws,nop,nop,ts,sok,eol+1:df,id+:0`,
}

func TestParseTCPSig(t *testing.T) {
	for _, s := range testSigs {
		sig, err := ParseTCPSignature("", s)
		if err != nil {
			t.Errorf("unable to parse `%v` -- %v", s, err)
		} else {
			t.Logf("%v", sig)
		}
	}
}
