// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"pkg/minigraph"
	log "pkg/minilog"
)

var (
	templates       *template.Template
	funcMap         template.FuncMap
	onceMap         map[string]bool
	current         string
	currentTemplate string
	data            map[string]string
	stopNode        bool
)

var (
	alphabet = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
)

// we have to return something with template functions...
func stop() string {
	stopNode = true
	return ""
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func csvSlice(s string) []string {
	log.Debug("csvSlice: %v", s)
	return strings.Split(s, ",")
}

func jsonMap(s string) map[string]string {
	log.Debug("jsonMap: %v", s)
	m := make(map[string]string)
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		log.Fatalln(err)
	}
	log.Debug("created map: %v", m)
	return m
}

func isEndpoint(n minigraph.Node) bool {
	return n.Type() == minigraph.TYPE_ENDPOINT
}

func isNetwork(n minigraph.Node) bool {
	return n.Type() == minigraph.TYPE_NETWORK
}

func setData(n minigraph.Node, key, value string) string {
	if !isEndpoint(n) {
		return ""
	}
	e := n.(*minigraph.Endpoint)
	e.D[key] = value
	_, err := dc.UpdateEndpoints(e)
	if err != nil {
		log.Fatalln(err)
	}
	return ""
}

func tDebug(format string, arg ...interface{}) string {
	log.Debug(format, arg...)
	f := "# debug " + format
	if log.WillLog(log.DEBUG) {
		return fmt.Sprintf(f, arg...)
	} else {
		return ""
	}
}

func tInfo(format string, arg ...interface{}) string {
	log.Info(format, arg...)
	f := "# info " + format
	if log.WillLog(log.INFO) {
		return fmt.Sprintf(f, arg...)
	} else {
		return ""
	}
}

func tError(format string, arg ...interface{}) string {
	log.Error(format, arg...)
	f := "# error " + format
	if log.WillLog(log.ERROR) {
		return fmt.Sprintf(f, arg...)
	} else {
		return ""
	}
}

func tWarn(format string, arg ...interface{}) string {
	log.Warn(format, arg...)
	f := "# warn " + format
	if log.WillLog(log.WARN) {
		return fmt.Sprintf(f, arg...)
	} else {
		return ""
	}
}

func tFatal(format string, arg ...interface{}) string {
	log.Fatal(format, arg...)
	return ""
}

func once() bool {
	if onceMap[current] {
		return false
	}
	return true
}

func set(k, v string) string {
	data[k] = v
	return ""
}

func get(k string) string {
	return data[k]
}

func init() {
	data = make(map[string]string)
	onceMap = make(map[string]bool)
	funcMap = template.FuncMap{
		"debug":      tDebug,
		"info":       tInfo,
		"error":      tError,
		"warn":       tWarn,
		"fatal":      tFatal,
		"once":       once,
		"isEndpoint": isEndpoint,
		"isNetwork":  isNetwork,
		"set":        set,
		"get":        get,
		"setData":    setData,
		"jsonMap":    jsonMap,
		"csvSlice":   csvSlice,
		"contains":   contains,
		"stop":       stop,
	}
}

func parseTemplates(path string) error {
	m := filepath.Join(path, "*.template")
	filenames, err := filepath.Glob(m)
	if err != nil {
		return err
	}

	sort.Strings(filenames)

	log.Debugln("ordered template list:")
	for _, v := range filenames {
		log.Debugln(v)
	}

	templates = template.New("").Funcs(funcMap)

	templates, err = templates.ParseFiles(filenames...)
	if err != nil {
		return err
	}

	log.Debug("parsed templates: %v", templates.DefinedTemplates())
	return nil
}

type parseObj struct {
	Config map[string]string
	Node   minigraph.Node
}

func parse(c map[string]string, g []minigraph.Node) ([]byte, error) {
	log.Debugln("parse")

	log.Debug("parsing %v nodes", len(g))

	t := templates.Templates()

	// it turns out templates with multiple files don't stay in their order
	// so we have to sort them here
	var tNames []string
	for _, v := range t {
		tNames = append(tNames, v.Name())
	}
	sort.Strings(tNames)

	var output bytes.Buffer

	for _, grp := range alphabet {
		log.Debug("group %v", grp)
		for _, n := range g {
			log.Debug("parsing node %v", n)
			po := parseObj{
				Config: c,
				Node:   n,
			}
			for _, v := range tNames {
				if !strings.HasPrefix(v, grp) {
					continue
				}
				tmpl := templates.Lookup(v)
				var o bytes.Buffer
				log.Debug("executing template %v on node %v", tmpl.Name(), n)
				current = tmpl.Name()
				err := tmpl.Execute(&o, po)
				if err != nil {
					return nil, err
				}
				onceMap[current] = true
				if strings.TrimSpace(o.String()) != "" {
					output.WriteString(fmt.Sprintf("\n### node %v ###\n", n.ID()))
					output.WriteString(pretty(o))
				}
				if stopNode {
					stopNode = false
					break
				}
			}
		}
	}

	return output.Bytes(), nil
}
