// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandia-minimega/discovery/v2/pkg/discovery"
	"github.com/sandia-minimega/discovery/v2/pkg/minigraph"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

type Data struct {
	Data string
}

type JuniperInterfaces []struct {
	Interface []struct {
		Name        Data
		Description []Data
		Unit        []struct {
			Name   Data
			Family []struct {
				Inet []struct {
					Address []struct {
						Name      Data
						Preferred []interface{} // may be omitted
					}
				}
				Inet6 []struct {
					Address []struct {
						Name      Data
						Preferred []interface{} // may be omitted
					}
				}
			}
		}
	}
}

type JuniperConfig struct {
	Interfaces JuniperInterfaces
	Groups     []struct {
		Name       Data
		Interfaces JuniperInterfaces
	}
	ApplyGroups []Data `json:"apply-groups"`
}

func processJuniper(dc *discovery.Client, ID int, interfaces JuniperInterfaces) error {
	for _, ifaces := range interfaces {
		for _, iface := range ifaces.Interface {
			log.Info("found interface: %v", iface.Name.Data)

			for _, unit := range iface.Unit {
				// assumption... only one ip/ipv6 per unit
				var ip, ip6 string

				for _, family := range unit.Family {
					for _, inet := range family.Inet {
						for _, addr := range inet.Address {
							log.Info("found ip address: %v", addr.Name.Data)

							v, _, err := net.ParseCIDR(addr.Name.Data)
							if err != nil || v.IsLoopback() {
								continue
							}

							// only update the IP if this one is the preferred/primary
							if ip == "" || len(addr.Preferred) > 0 {
								ip = addr.Name.Data
							}
						}
					}

					for _, inet6 := range family.Inet6 {
						for _, addr := range inet6.Address {
							log.Info("found ipv6 address: %v", addr.Name.Data)

							v, _, err := net.ParseCIDR(addr.Name.Data)
							if err != nil || v.IsLoopback() {
								continue
							}

							// only update the IP if this one is the preferred/primary
							if ip6 == "" || len(addr.Preferred) > 0 {
								ip6 = addr.Name.Data
							}
						}
					}
				}

				desc := iface.Name.Data + ":" + unit.Name.Data

				if err := AddInterface(dc, ID, desc, ip, ip6); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func parseJuniper(f *os.File, dc *discovery.Client) error {
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `show configuration | display json`) {
			break
		}
	}

	// Should be JSON until line starting with comment
	var data []byte
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "#") {
			break
		}

		data = append(data, scanner.Bytes()...)
		data = append(data, 10) // new line
	}

	var configs struct {
		Configuration []JuniperConfig
	}
	if err := json.Unmarshal(data, &configs); err != nil {
		syntax, ok := err.(*json.SyntaxError)
		if !ok {
			return err
		}

		js := string(data)

		start, end := strings.LastIndex(js[:syntax.Offset], "\n")+1, len(js)
		if idx := strings.Index(js[start:], "\n"); idx >= 0 {
			end = start + idx
		}

		line, pos := strings.Count(js[:start], "\n"), int(syntax.Offset)-start-1

		fmt.Printf("Error in line %d: %s \n", line, err)
		fmt.Printf("%s\n%s^", js[start:end], strings.Repeat(" ", pos))

		return err
	}

	e := &minigraph.Endpoint{
		D: map[string]string{
			"router": "true",
			"type":   "juniper",
			"name":   filepath.Base(f.Name()),
			"icon":   "router",
		},
	}

	var ID int

	if !*f_dryrun {
		es, err := dc.InsertEndpoints(e)
		e = es[0]
		if err != nil {
			return err
		}

		ID = es[0].ID()
	}

	for _, config := range configs.Configuration {
		processJuniper(dc, ID, config.Interfaces)

		// not really how groups work but good enough for now
		for _, apply := range config.ApplyGroups {
			for _, group := range config.Groups {
				if apply.Data == group.Name.Data {
					processJuniper(dc, ID, group.Interfaces)
				}
			}
		}
	}

	return nil
}
