/*
Package client is the Landns client library for golang.

This package is a simple helper for use package github.com/macrat/landns/lib-landns in a client.
Please see also lib-landns document ( https://godoc.org/github.com/macrat/landns/lib-landns ).
*/
package client

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/macrat/landns/lib-landns"
)

// Client is the instance for operate dynamic records.
type Client struct {
	Endpoint *url.URL // The Landns API endpoint URL.
	client   *http.Client
}

// New is make new Client instance.
func New(endpoint *url.URL) Client {
	return Client{
		Endpoint: endpoint,
		client:   &http.Client{},
	}
}

func (c Client) do(method, path string, body fmt.Stringer) (response landns.DynamicRecordSet, err error) {
	u, err := c.Endpoint.Parse(path)
	if err != nil {
		return
	}

	us := u.String()
	if strings.HasSuffix(us, "/") {
		us = us[:len(us)-1]
	}

	var r io.Reader
	if body != nil {
		r = strings.NewReader(body.String())
	}
	req, err := http.NewRequest(method, us, r)
	if err != nil {
		return
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return
	}

	return response, response.UnmarshalText(rbody)
}

// Set do send and register records.
func (c Client) Set(records landns.DynamicRecordSet) error {
	_, err := c.do("POST", "", records)
	return err
}

// Remove will remove one record from Landns server.
func (c Client) Remove(id int) error {
	_, err := c.do("DELETE", fmt.Sprintf("id/%d", id), nil)
	return err
}

// Get will receive all records from Landns server.
func (c Client) Get() (landns.DynamicRecordSet, error) {
	return c.do("GET", "", nil)
}

// Glob will receive records that match with given query.
func (c Client) Glob(query string) (landns.DynamicRecordSet, error) {
	return c.do("GET", fmt.Sprintf("glob/%s", query), nil)
}
