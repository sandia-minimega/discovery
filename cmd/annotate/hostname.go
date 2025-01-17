// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bufio"
	"errors"
	"flag"
	"math/rand"
	"os"
	"regexp"
	"strings"

	"github.com/sandia-minimega/discovery/v2/pkg/commands"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

type CommandHostname struct {
	commands.Base // embed

	words, domain string
}

func init() {
	base := commands.Base{
		Flags: flag.NewFlagSet("hostname", flag.ExitOnError),
		Usage: "hostname [OPTION]...",
		Short: "annotate with random hostnames",
		Long: `
Uses a wordlist to generate hostnames for nodes. All hostnames will be in the
.com domain unless otherwise specified.
`,
	}

	cmd := &CommandHostname{Base: base}
	cmd.Flags.StringVar(&cmd.words, "words", "/usr/share/dict/words", "word file")
	cmd.Flags.StringVar(&cmd.domain, "domain", ".com", "domain for new hostnames")

	commands.Append(cmd)
}

func (c *CommandHostname) Run() error {
	words, err := readWords(c.words, rng)
	if err != nil {
		return err
	}

	endpoints, err := dc.GetEndpoints("", "")
	if err != nil {
		return err
	}

	// hostnames (sans domain) that are already in use
	taken := map[string]bool{}

	for _, v := range endpoints {
		if s, ok := v.D["hostname"]; ok {
			for _, v := range strings.Split(s, ",") {
				taken[strings.TrimSuffix(v, c.domain)] = true
			}
		}
	}

	// current index in words
	var i int

	for _, v := range endpoints {
		if _, ok := v.D["hostname"]; ok && !*f_overwrite {
			continue
		}

		// find the next untaken hostname (ignore if overwriting)
		for ; i < len(words); i++ {
			if !taken[words[i]] || *f_overwrite {
				break
			}
		}
		if i >= len(words) {
			return errors.New("ran out of words")
		}

		v.D["hostname"] = words[i] + c.domain
		log.Debug("updating node %v with hostname %v", v, v.D["hostname"])
		i++

		if err := UpdateEndpoint(v); err != nil {
			return err
		}
	}

	return nil
}

func readWords(fname string, rng *rand.Rand) ([]string, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var words []string

	// only want alpha chars
	valid := regexp.MustCompile(`^[a-z]+$`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		if valid.MatchString(s) {
			words = append(words, s)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// shuffle words with Fisher-Yates algorithm
	for i := len(words) - 1; i > 0; i-- {
		j := rng.Intn(i)
		words[i], words[j] = words[j], words[i]
	}

	return words, nil
}
