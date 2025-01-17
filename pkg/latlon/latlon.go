// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package latlon

import (
	"errors"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"

	"github.com/oschwald/geoip2-golang"
	"github.com/oschwald/maxminddb-golang"
	log "github.com/sandia-minimega/discovery/v2/pkg/minilog"
)

type Result struct {
	geoip2.City // embed
	geoip2.ISP  // embed
}

type DB struct {
	readers []*maxminddb.Reader
}

func Open(d string) (*DB, error) {
	log.Debug("opening GeoIP databases in %v", d)

	files, err := ioutil.ReadDir(d)
	if err != nil {
		return nil, err
	}

	db := &DB{}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".mmdb") {
			log.Debug("skipping %v", f.Name())
			continue
		}

		r, err := maxminddb.Open(filepath.Join(d, f.Name()))
		if err != nil {
			db.Close()
			return nil, err
		}

		db.readers = append(db.readers, r)
	}

	if len(db.readers) == 0 {
		return nil, errors.New("no mmdb files found")
	}

	return db, nil
}

func (db *DB) Close() {
	for _, r := range db.readers {
		r.Close()
	}
}

func (db *DB) Lookup(ip net.IP) (*Result, error) {
	log.Debug("lookup: %v", ip)

	var res Result
	for _, r := range db.readers {

		switch r.Metadata.DatabaseType {
		case "GeoIP2-ISP":
			if err := r.Lookup(ip, &res.ISP); err != nil {
				return nil, err
			}
		case "GeoIP2-City":
			if err := r.Lookup(ip, &res.City); err != nil {
				return nil, err
			}
		case "GeoIP2-Country":
			// ignore, city is more specific
		}

	}

	return &res, nil
}
