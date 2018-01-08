package haci

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"strings"

	"gopkg.in/jmcvetta/napping.v3"
)

type Network struct {
	ID          string   `json:"ID"`
	CreateDate  string   `json:"createDate"`
	CreateFrom  string   `json:"createFrom"`
	Description string   `json:"description"`
	Network     string   `json:"network"`
	Tags        []string `json:"tags"`
}

func (n Network) IP() (string, error) {
	ip, _, err := net.ParseCIDR(n.Network)

	if err != nil {
		return "", err
	}

	return ip.String(), nil
}

type Client interface {
	Get(network string) (Network, error)
	List(network string) ([]Network, error)
	Assign(network string, description string, cidr int, tags []string) (Network, error)
	Delete(network string) error
}

type WebClient struct {
	napping napping.Session
	URL     string
	Root    string
}

// A very simple and limited client for unit tests.
type FakeClient struct {
	Networks map[string]Network
	Counter  int
}

func NewWebClient(url, username, password, root string) (haci *WebClient, err error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	haci = &WebClient{
		napping: napping.Session{
			Log:      false,
			Client:   client,
			Userinfo: neturl.UserPassword(username, password),
		},
		URL:  strings.TrimRight(url, "/"),
		Root: root,
	}
	return
}

func NewFakeClient() *FakeClient {
	return &FakeClient{Networks: map[string]Network{}, Counter: 1}
}

func (c *WebClient) Get(network string) (network1 Network, err error) {
	resp, err := c.napping.Get(c.URL+"/RESTWrapper/getNetworkDetails",
		&neturl.Values{
			"rootName": {c.Root},
			"network":  {network},
		},
		&network1,
		nil)

	if err != nil {
		return Network{}, err
	}

	if resp.Status() != 200 {
		return Network{}, fmt.Errorf("lookup failed: %s", resp.RawText())
	}

	return
}

func (c *WebClient) List(network string) (networks []Network, err error) {
	resp, err := c.napping.Get(c.URL+"/RESTWrapper/getSubnets",
		&neturl.Values{
			"rootName": {c.Root},
			"supernet": {network},
		},
		&networks,
		nil)

	if err != nil {
		return []Network{}, err
	}

	if resp.Status() != 200 {
		return []Network{}, fmt.Errorf("list failed: %s", resp.RawText())
	}

	return
}

func (c *WebClient) Assign(network, description string, cidr int, tags []string) (network1 Network, err error) {
	resp, err := c.napping.Get(c.URL+"/RESTWrapper/assignFreeSubnet",
		&neturl.Values{
			"rootName":    {c.Root},
			"supernet":    {network},
			"description": {description},
			"cidr:":       {string(cidr)},
		},
		&network1,
		nil)

	if err != nil {
		return Network{}, err
	}

	if resp.Status() != 200 {
		return Network{}, fmt.Errorf("assignment failed: %s", resp.RawText())
	}

	return
}

func (c *WebClient) Delete(network string) (err error) {
	resp, err := c.napping.Get(c.URL+"/RESTWrapper/delNet",
		&neturl.Values{
			"rootName": {c.Root},
			"network":  {network},
		},
		nil,
		nil)

	if err != nil {
		return err
	}

	if resp.Status() != 200 {
		return fmt.Errorf("delete failed: %s", resp.RawText())
	}

	return
}

func (c *FakeClient) Get(network string) (Network, error) {
	return c.Networks[network], nil
}

func (c *FakeClient) List(network string) (networks []Network, err error) {
	for _, n := range c.Networks {
		networks = append(networks, n)
	}

	return
}

func (c *FakeClient) Assign(network, description string, cidr int, tags []string) (network1 Network, err error) {

	ip := fmt.Sprintf("10.0.0.%d/32", c.Counter)

	network1 = Network{
		ID:          fmt.Sprintf("%d", c.Counter),
		Network:     ip,
		Description: description,
		Tags:        tags,
	}

	c.Networks[ip] = network1

	c.Counter = c.Counter + 1

	return
}

func (c *FakeClient) Delete(network string) (err error) {
	delete(c.Networks, network)

	return
}
