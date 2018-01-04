package haci

import (
	"crypto/tls"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"

	"github.com/nu7hatch/gouuid"
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

type Client interface {
	Get(string) (Network, error)
	List(string) ([]Network, error)
	Assign(string, string, int, []string) (Network, error)
	Delete(string) error
}

type WebClient struct {
	napping napping.Session
	URL     string
	Root    string
}

// A very simple and limited client for unit tests.
type FakeClient struct {
	Networks map[string]Network
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
	return &FakeClient{Networks: map[string]Network{}}
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
	u, _ := uuid.NewV4()

	network1 = Network{
		ID:          u.String(),
		Network:     u.String(),
		Description: description,
		Tags:        tags,
	}

	c.Networks[u.String()] = network1

	return
}

func (c *FakeClient) Delete(network string) (err error) {
	delete(c.Networks, network)

	return
}
