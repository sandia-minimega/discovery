// Copyright 2018 National Technology & Engineering Solutions of Sandia, LLC
// (NTESS). Under the terms of Contract DE-NA0003525 with NTESS, the U.S.
// Government retains certain rights in this software.

package discovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"

	"pkg/minigraph"
	log "pkg/minilog"
)

const (
	Port      = 8000
	EDGE_NONE = -1
)

type jsonError struct {
	Request string `json:"request"`
	Error   string `json:"error"`
}

func WriteError(w io.Writer, r *http.Request, err error) {
	e := &jsonError{
		Request: fmt.Sprintf("%v\t%v", r.Method, r.RequestURI),
		Error:   err.Error(),
	}
	b, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		log.Fatalln(err)
	}
	_, err = w.Write(b)
	if err != nil {
		log.Fatalln(err)
	}
}

func ReadError(i io.Reader) error {
	var e jsonError
	d := json.NewDecoder(i)
	err := d.Decode(&e)
	if err != nil {
		log.Fatalln(err)
	}
	return fmt.Errorf("%v : %v", e.Request, e.Error)
}

type Client struct {
	server string
}

func New(s string) *Client {
	server := s
	_, _, err := net.SplitHostPort(s)
	if err != nil {
		server = net.JoinHostPort(s, fmt.Sprintf("%v", Port))
	}
	log.Debug("using server %v", server)
	return &Client{
		server: fmt.Sprintf("http://%v", server),
	}
}

// GetEndpoint wraps GetEndpoints and returns the first node that matches or an
// error if more than one node matches.
func (c *Client) GetEndpoint(k, v string) (*minigraph.Endpoint, error) {
	res, err := c.GetEndpoints(k, v)
	if err != nil {
		return nil, err
	}

	switch len(res) {
	case 0:
		return nil, errors.New("endpoint not found")
	case 1:
		return res[0], nil
	default:
		return nil, errors.New("more than one endpoint found")
	}
}

// GetEndpoints returns endpoints by search terms, or all endpoints. If k,v are
// empty, return all endpoints. If only v is set, do a freeform search on all
// endpoints, if k is set, search for v on the key k. Endpoints will be sorted
// by ID.
func (c *Client) GetEndpoints(k, v string) ([]*minigraph.Endpoint, error) {
	var path string
	if k == "" && v == "" {
		path = fmt.Sprintf("%v/endpoints/", c.server)
	} else if k == "" {
		path = fmt.Sprintf("%v/endpoints/%v", c.server, v)
	} else {
		path = fmt.Sprintf("%v/endpoints/%v/%v", c.server, k, v)
	}

	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret []*minigraph.Endpoint
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}

	// sort so that they can be processed in a consistent order
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].NID < ret[j].NID
	})

	return ret, nil
}

func (c *Client) GetConfig() (map[string]string, error) {
	path := fmt.Sprintf("%v/config/", c.server)

	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret = make(map[string]string)
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Client) SetConfig(k, v string) error {
	httpClient := &http.Client{}

	body := bytes.NewBufferString(v)

	path := fmt.Sprintf("%v/config/%v", c.server, k)

	httpRequest, err := http.NewRequest(http.MethodPost, path, body)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err := ReadError(resp.Body)
		return err
	}

	return nil
}

func (c *Client) DeleteConfig(k string) error {
	httpClient := &http.Client{}

	var path string
	path = fmt.Sprintf("%v/config/%v", c.server, k)

	httpRequest, err := http.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return err
	}

	return nil
}

// Get networks by search terms, or all networks. If k,v are empty, return all
// networks. If only v is set, do a freeform search on all networks, if k is
// set, search for v on the key k. Networks will be sorted by ID.
func (c *Client) GetNetworks(k, v string) ([]*minigraph.Network, error) {
	var path string
	if k == "" && v == "" {
		path = fmt.Sprintf("%v/networks/", c.server)
	} else if k == "" {
		path = fmt.Sprintf("%v/networks/%v", c.server, v)
	} else {
		path = fmt.Sprintf("%v/networks/%v/%v", c.server, k, v)
	}

	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret []*minigraph.Network
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}

	// sort so that they can be processed in a consistent order
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].NID < ret[j].NID
	})

	return ret, nil
}

func (c *Client) InsertEndpoints(e ...*minigraph.Endpoint) ([]*minigraph.Endpoint, error) {
	httpClient := &http.Client{}

	b, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		log.Fatalln(err)
	}

	body := bytes.NewReader(b)

	path := fmt.Sprintf("%v/endpoints/", c.server)

	httpRequest, err := http.NewRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret []*minigraph.Endpoint
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Client) UpdateEndpoints(e ...*minigraph.Endpoint) ([]*minigraph.Endpoint, error) {
	httpClient := &http.Client{}

	b, err := json.MarshalIndent(e, "", "    ")
	if err != nil {
		log.Fatalln(err)
	}

	body := bytes.NewReader(b)

	path := fmt.Sprintf("%v/endpoints/", c.server)

	httpRequest, err := http.NewRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret []*minigraph.Endpoint
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil

}

func (c *Client) InsertNetworks(n ...*minigraph.Network) ([]*minigraph.Network, error) {
	httpClient := &http.Client{}

	b, err := json.MarshalIndent(n, "", "    ")
	if err != nil {
		log.Fatalln(err)
	}

	body := bytes.NewReader(b)

	path := fmt.Sprintf("%v/networks/", c.server)

	httpRequest, err := http.NewRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret []*minigraph.Network
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Client) UpdateNetworks(n ...*minigraph.Network) ([]*minigraph.Network, error) {
	httpClient := &http.Client{}

	b, err := json.MarshalIndent(n, "", "    ")
	if err != nil {
		log.Fatalln(err)
	}

	body := bytes.NewReader(b)
	if err != nil {
		log.Fatalln(err)
	}

	path := fmt.Sprintf("%v/networks/", c.server)

	httpRequest, err := http.NewRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret []*minigraph.Network
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Client) DeleteEndpoints(k, v string) ([]*minigraph.Endpoint, error) {
	httpClient := &http.Client{}

	var path string
	if k == "" {
		path = fmt.Sprintf("%v/endpoints/%v", c.server, v)
	} else {
		path = fmt.Sprintf("%v/endpoints/%v/%v", c.server, k, v)
	}

	httpRequest, err := http.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret []*minigraph.Endpoint
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) DeleteNetworks(k, v string) ([]*minigraph.Network, error) {
	httpClient := &http.Client{}

	var path string
	if k == "" {
		path = fmt.Sprintf("%v/networks/%v", c.server, v)
	} else {
		path = fmt.Sprintf("%v/networks/%v/%v", c.server, k, v)
	}

	httpRequest, err := http.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret []*minigraph.Network
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) Neighbors(k, v string) ([]minigraph.Node, error) {
	var path string
	if k == "" && v == "" {
		path = fmt.Sprintf("%v/neighbors/", c.server)
	} else if k == "" {
		path = fmt.Sprintf("%v/neighbors/%v", c.server, v)
	} else {
		path = fmt.Sprintf("%v/neighbors/%v/%v", c.server, k, v)
	}

	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret []minigraph.Node
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *Client) Save(path string) error {
	url := fmt.Sprintf("%v/daemon/save/%v", c.server, path)
	log.Debug("using url: %v", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return err
	}

	return nil
}

func (c *Client) Load(path string) error {
	url := fmt.Sprintf("%v/daemon/load/%v", c.server, path)
	log.Debug("using url: %v", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return err
	}

	return nil
}

func (c *Client) Connect(nnid, enid int, eidx int) (*minigraph.Endpoint, error) {
	var url string
	if eidx == EDGE_NONE {
		url = fmt.Sprintf("%v/connect/%v/%v", c.server, nnid, enid)
	} else {
		url = fmt.Sprintf("%v/connect/%v/%v/%v", c.server, nnid, enid, eidx)
	}

	resp, err := http.Post(url, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret *minigraph.Endpoint
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) Disconnect(nnid, enid int) (*minigraph.Endpoint, error) {
	url := fmt.Sprintf("%v/disconnect/%v/%v", c.server, nnid, enid)

	resp, err := http.Post(url, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := ReadError(resp.Body)
		return nil, err
	}

	var ret *minigraph.Endpoint
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
