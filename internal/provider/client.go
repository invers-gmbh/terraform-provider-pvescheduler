package provider

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type PveClient struct {
	Endpoint           string
	InsecureSkipVerify bool
	Nodes              []string

	apiToken  string
	ticket    string
	csrfToken string
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

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		},
	}

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

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify},
		},
	}
	resp, err := httpClient.Do(req)
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
