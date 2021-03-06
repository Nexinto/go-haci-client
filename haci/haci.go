package haci

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"strings"

	ccidr "github.com/apparentlymart/go-cidr/cidr"
	"gopkg.in/jmcvetta/napping.v3"
)

type Network struct {
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
	List(supernet string) ([]Network, error)
	Assign(supernet string, description string, cidr int, tags []string) (Network, error)
	Delete(network string) error
	Add(network, description string, tags []string) error
	Search(description string, exact bool) ([]Network, error)
	Reset() error
	String() string
}

type WebClient struct {
	napping napping.Session
	URL     string
	Root    string
}

// A very simple and limited client for unit tests.
type FakeClient struct {
	UseFirst  bool
	Supernets map[string]*FakeSupernet
	Added     map[string]Network
}

type FakeSupernet struct {
	Networks map[string]Network
	Network  net.IPNet
	Last     net.IP
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

// Create a new HaCi fake client.
func NewFakeClient() *FakeClient {
	return &FakeClient{Supernets: map[string]*FakeSupernet{}, Added: map[string]Network{}}
}

// Create a new HaCi fake client that assigns the first (network address) of a network.
func NewFakeClientUsesFirst() *FakeClient {
	return &FakeClient{Supernets: map[string]*FakeSupernet{}, Added: map[string]Network{}, UseFirst: true}
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

func (c *WebClient) List(supernet string) (networks []Network, err error) {
	resp, err := c.napping.Get(c.URL+"/RESTWrapper/getSubnets",
		&neturl.Values{
			"rootName": {c.Root},
			"supernet": {supernet},
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

func (c *WebClient) Assign(supernet, description string, cidr int, tags []string) (network1 Network, err error) {
	resp, err := c.napping.Get(c.URL+"/RESTWrapper/assignFreeSubnet",
		&neturl.Values{
			"rootName":    {c.Root},
			"supernet":    {supernet},
			"description": {description},
			"cidr":        {fmt.Sprintf("%d", cidr)},
			"tags":        {strings.Join(tags, " ")},
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
			"rootName":    {c.Root},
			"network":     {network},
			"networkLock": {"1"},
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

func (c *WebClient) Add(network, description string, tags []string) error {
	resp, err := c.napping.Get(c.URL+"/RESTWrapper/addNet",
		&neturl.Values{
			"rootName":    {c.Root},
			"network":     {network},
			"description": {description},
			"tags":        {strings.Join(tags, " ")},
		},
		nil,
		nil)

	if err != nil {
		return err
	}

	if resp.Status() != 200 {
		return fmt.Errorf("assignment failed: %s", resp.RawText())
	}

	return nil
}

func (c *WebClient) Search(description string, exact bool) (networks []Network, err error) {
	values := neturl.Values{
		"rootName":    {c.Root},
		"search":      {description},
		"withDetails": {"1"},
	}
	if exact {
		values["exact"] = []string{"true"}
	}
	resp, err := c.napping.Get(c.URL+"/RESTWrapper/search", &values, &networks, nil)

	if err != nil {
		return []Network{}, err
	}

	if resp.Status() != 200 {
		return []Network{}, fmt.Errorf("search failed: %s", resp.RawText())
	}

	return

}

func (c *WebClient) Reset() error {
	return fmt.Errorf("Reset() not implemented in haci.WebClient")
}

func (c *FakeClient) Get(network string) (Network, error) {
	if n, ok := c.Added[network]; ok {
		return n, nil
	}

	for _, s := range c.Supernets {
		if n, ok := s.Networks[network]; ok {
			return n, nil
		}
	}
	return Network{}, fmt.Errorf("network %s not found", network)
}

func (c *WebClient) String() string {
	return fmt.Sprintf("HaCi at %s(%s)", c.URL, c.Root)
}

func (c *FakeClient) List(supernet string) (networks []Network, err error) {
	if s, ok := c.Supernets[supernet]; ok {
		for _, n := range s.Networks {
			networks = append(networks, n)
		}
	}

	return
}

func (c *FakeClient) Assign(supernet, description string, cidr int, tags []string) (network1 Network, err error) {

	ip, net, err := net.ParseCIDR(supernet)
	if err != nil {
		return Network{}, err
	}

	if _, ok := c.Supernets[supernet]; !ok {
		last := ip
		if c.UseFirst {
			last = ccidr.Dec(last)
		}
		c.Supernets[supernet] = &FakeSupernet{Network: *net, Networks: map[string]Network{}, Last: last}
	}

	_, l := ccidr.AddressRange(net)
	if l.Equal(c.Supernets[supernet].Last) {
		return Network{}, fmt.Errorf("out of addresses in %s", supernet)
	}

	newip := ccidr.Inc(c.Supernets[supernet].Last)
	netname := fmt.Sprintf("%s/32", newip.String())

	network1 = Network{
		Network:     netname,
		Description: description,
		Tags:        tags,
	}

	c.Supernets[supernet].Networks[netname] = network1
	c.Supernets[supernet].Last = newip

	return
}

func (c *FakeClient) Delete(network string) error {
	for _, s := range c.Supernets {
		delete(s.Networks, network)
	}
	delete(c.Added, network)
	return nil
}

func (c *FakeClient) Add(network, description string, tags []string) error {
	for _, s := range c.Supernets {
		if _, exists := s.Networks[network]; exists {
			return fmt.Errorf("network %s already exists", network)
		}
	}
	if _, exists := c.Added[network]; exists {
		return fmt.Errorf("network %s already exists", network)
	}
	c.Added[network] = Network{Network: network, Description: description, Tags: tags}
	return nil
}

func (c *FakeClient) Search(description string, exact bool) (networks []Network, err error) {
	for _, n := range c.Added {
		if exact && n.Description == description || !exact && strings.Contains(n.Description, description) {
			networks = append(networks, n)
		}
	}

	for _, s := range c.Supernets {
		for _, n := range s.Networks {
			if exact && n.Description == description || !exact && strings.Contains(n.Description, description) {
				networks = append(networks, n)
			}
		}
	}
	return
}

func (c *FakeClient) Reset() error {
	c.Supernets = map[string]*FakeSupernet{}
	c.Added = map[string]Network{}

	return nil
}

func (c *FakeClient) String() string {
	return "HaCi fake client"
}
