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
	MaxCpu float64 `json:"maxcpu"`
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
	endpoint_url := fmt.Sprintf("%s/api2/json/access/ticket", endpoint)

	body := url.Values{}
	body.Set("username", username)
	body.Set("password", password)
	req, err := http.NewRequest("POST", endpoint_url, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := newHTTPClient(insecure)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PVE API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Ticket              string `json:"ticket"`
			CSRFPreventionToken string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &PveClient{
		Endpoint:           endpoint,
		InsecureSkipVerify: insecure,
		Nodes:              nodes,
		http:               httpClient,
		ticket:             result.Data.Ticket,
		csrfToken:          result.Data.CSRFPreventionToken,
	}, nil
}

func (c *PveClient) addAuth(req *http.Request) {
	if c.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.apiToken))
	} else {
		req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: c.ticket})
		req.Header.Set("CSRFPreventionToken", c.csrfToken)
	}
}

func (c *PveClient) GetNodes() ([]PveNode, error) {
	url := fmt.Sprintf("%s/api2/json/nodes", c.Endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
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
