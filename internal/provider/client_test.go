package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddAuth_Token(t *testing.T) {
	c := &PveClient{apiToken: "user@pam!mytoken=abc123"}
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	c.addAuth(req)
	if got := req.Header.Get("Authorization"); got != "PVEAPIToken=user@pam!mytoken=abc123" {
		t.Errorf("unexpected Authorization header: %q", got)
	}
}

func TestAddAuth_Ticket(t *testing.T) {
	c := &PveClient{ticket: "PVE:user:AABBCC", csrfToken: "tok123"}
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	c.addAuth(req)
	if req.Header.Get("CSRFPreventionToken") != "tok123" {
		t.Errorf("missing CSRF header")
	}
	cookies := req.Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "PVEAuthCookie" && cookie.Value == "PVE:user:AABBCC" {
			found = true
		}
	}
	if !found {
		t.Errorf("PVEAuthCookie not set")
	}
}

func TestGetNodes_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []PveNode{
				{Node: "pve01", Status: "online", Mem: 4e9, MaxMem: 8e9, Cpu: 0.2},
				{Node: "pve02", Status: "online", Mem: 2e9, MaxMem: 8e9, Cpu: 0.1},
			},
		})
	}))
	defer srv.Close()

	c := NewClientWithToken(srv.URL, "tok", false, nil)
	nodes, err := c.GetNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Node != "pve01" {
		t.Errorf("unexpected first node: %s", nodes[0].Node)
	}
}

func TestGetNodes_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClientWithToken(srv.URL, "bad-token", false, nil)
	_, err := c.GetNodes()
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestNewClientWithPassword_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api2/json/access/ticket" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{
				"ticket":              "PVE:user@pam:ABC123",
				"CSRFPreventionToken": "csrf-tok",
			},
		})
	}))
	defer srv.Close()

	c, err := NewClientWithPassword(srv.URL, "user@pam", "pass", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.ticket != "PVE:user@pam:ABC123" {
		t.Errorf("unexpected ticket: %q", c.ticket)
	}
	if c.csrfToken != "csrf-tok" {
		t.Errorf("unexpected CSRF token: %q", c.csrfToken)
	}
}

func TestNewClientWithPassword_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := NewClientWithPassword(srv.URL, "user@pam", "wrong", false, nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
