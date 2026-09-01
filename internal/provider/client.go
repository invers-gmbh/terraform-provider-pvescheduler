package provider

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const requestTimeout = 30 * time.Second

type PveClient struct {
	Endpoint           string
	InsecureSkipVerify bool
	Nodes              []string

	http      *http.Client
	apiToken  string
	username  string
	password  string
	ticket    string
	csrfToken string
}

func newHTTPClient(insecure bool) *http.Client {
	return &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		},
	}
}

type PveNode struct {
	Node   string  `json:"node"`
	Status string  `json:"status"`
	Mem    float64 `json:"mem"`
	MaxMem float64 `json:"maxmem"`
	Cpu    float64 `json:"cpu"`
}

func NewClientWithToken(endpoint, apiToken string, insecure bool, nodes []string) *PveClient {
	return &PveClient{
		Endpoint:           endpoint,
		Nodes:              nodes,
		apiToken:           apiToken,
		InsecureSkipVerify: insecure,
		http:               newHTTPClient(insecure),
	}
}

func NewClientWithPassword(endpoint, username, password string, insecure bool, nodes []string) (*PveClient, error) {
	c := &PveClient{
		Endpoint:           endpoint,
		InsecureSkipVerify: insecure,
		Nodes:              nodes,
		http:               newHTTPClient(insecure),
		username:           username,
		password:           password,
	}

	if err := c.authenticate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *PveClient) authenticate() error {
	endpointURL := fmt.Sprintf("%s/api2/json/access/ticket", c.Endpoint)

	body := url.Values{}
	body.Set("username", c.username)
	body.Set("password", c.password)
	req, err := http.NewRequest("POST", endpointURL, strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PVE API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Ticket              string `json:"ticket"`
			CSRFPreventionToken string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.ticket = result.Data.Ticket
	c.csrfToken = result.Data.CSRFPreventionToken
	return nil
}

func (c *PveClient) canReauthenticate() bool {
	return c.apiToken == "" && c.username != "" && c.password != ""
}

func (c *PveClient) addAuth(req *http.Request) {
	if c.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.apiToken))
	} else {
		req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: c.ticket})
		req.Header.Set("CSRFPreventionToken", c.csrfToken)
	}
}

func (c *PveClient) getNodesOnce() (*http.Response, error) {
	url := fmt.Sprintf("%s/api2/json/nodes", c.Endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)
	return c.http.Do(req)
}

func (c *PveClient) GetNodes() ([]PveNode, error) {
	resp, err := c.getNodesOnce()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized && c.canReauthenticate() {
		_ = resp.Body.Close()
		if err := c.authenticate(); err != nil {
			return nil, fmt.Errorf("PVE ticket expired and re-authentication failed: %w", err)
		}
		resp, err = c.getNodesOnce()
		if err != nil {
			return nil, err
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PVE API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data []PveNode `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
