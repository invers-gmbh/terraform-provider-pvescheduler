package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestNewClientWithToken_SetsTimeout(t *testing.T) {
	c := NewClientWithToken("http://example.com", "tok", false, nil)
	if c.http == nil {
		t.Fatal("expected an HTTP client")
	}
	if c.http.Timeout != requestTimeout {
		t.Errorf("expected timeout %v, got %v", requestTimeout, c.http.Timeout)
	}
}

func TestGetNodes_ReusesClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []PveNode{}})
	}))
	defer srv.Close()

	c := NewClientWithToken(srv.URL, "tok", false, nil)
	first := c.http
	if _, err := c.GetNodes(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetNodes(); err != nil {
		t.Fatal(err)
	}
	if c.http != first {
		t.Error("expected the same HTTP client across calls")
	}
}

func TestGetNodes_TimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	c := NewClientWithToken(srv.URL, "tok", false, nil)
	c.http.Timeout = 20 * time.Millisecond
	if _, err := c.GetNodes(); err == nil {
		t.Fatal("expected a timeout error")
	}
}

func TestNewClientWithPassword_SetsTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"ticket": "t", "CSRFPreventionToken": "c"},
		})
	}))
	defer srv.Close()

	c, err := NewClientWithPassword(srv.URL, "user@pam", "pass", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.http == nil || c.http.Timeout != requestTimeout {
		t.Error("expected a client with a timeout")
	}
}

func TestGetNodes_ReauthenticatesOnExpiredTicket(t *testing.T) {
	issued := 0
	nodeCalls := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/access/ticket", func(w http.ResponseWriter, r *http.Request) {
		issued++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{
				"ticket":              fmt.Sprintf("ticket-%d", issued),
				"CSRFPreventionToken": fmt.Sprintf("csrf-%d", issued),
			},
		})
	})
	mux.HandleFunc("/api2/json/nodes", func(w http.ResponseWriter, r *http.Request) {
		nodeCalls++
		cookie, err := r.Cookie("PVEAuthCookie")
		if err != nil || cookie.Value != "ticket-2" {
			http.Error(w, "ticket expired", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []PveNode{{Node: "pve01", Status: "online", Mem: 1e9, MaxMem: 8e9}},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClientWithPassword(srv.URL, "user@pam", "pass", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.ticket != "ticket-1" {
		t.Fatalf("expected initial ticket-1, got %q", c.ticket)
	}

	nodes, err := c.GetNodes()
	if err != nil {
		t.Fatalf("expected re-authentication to recover, got %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
	if c.ticket != "ticket-2" {
		t.Errorf("expected refreshed ticket-2, got %q", c.ticket)
	}
	if c.csrfToken != "csrf-2" {
		t.Errorf("expected refreshed csrf-2, got %q", c.csrfToken)
	}
	if issued != 2 {
		t.Errorf("expected 2 tickets issued, got %d", issued)
	}
	if nodeCalls != 2 {
		t.Errorf("expected 2 node calls, got %d", nodeCalls)
	}
}

func TestGetNodes_TokenAuthDoesNotRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClientWithToken(srv.URL, "tok", false, nil)
	if _, err := c.GetNodes(); err == nil {
		t.Fatal("expected error for 401 response")
	}
	if calls != 1 {
		t.Errorf("token auth should not retry, got %d calls", calls)
	}
}

func TestGetNodes_RetriesOnlyOnce(t *testing.T) {
	nodeCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/access/ticket", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"ticket": "t", "CSRFPreventionToken": "c"},
		})
	})
	mux.HandleFunc("/api2/json/nodes", func(w http.ResponseWriter, r *http.Request) {
		nodeCalls++
		http.Error(w, "still unauthorized", http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClientWithPassword(srv.URL, "user@pam", "pass", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetNodes(); err == nil {
		t.Fatal("expected error when re-authentication does not help")
	}
	if nodeCalls != 2 {
		t.Errorf("expected exactly 2 node calls, got %d", nodeCalls)
	}
}

func TestGetNodes_ReauthFailurePropagates(t *testing.T) {
	tickets := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/access/ticket", func(w http.ResponseWriter, r *http.Request) {
		tickets++
		if tickets > 1 {
			http.Error(w, "credentials revoked", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{"ticket": "t", "CSRFPreventionToken": "c"},
		})
	})
	mux.HandleFunc("/api2/json/nodes", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "ticket expired", http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := NewClientWithPassword(srv.URL, "user@pam", "pass", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetNodes()
	if err == nil {
		t.Fatal("expected error when re-authentication fails")
	}
	if !strings.Contains(err.Error(), "re-authentication failed") {
		t.Errorf("expected re-authentication error, got %v", err)
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
