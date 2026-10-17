package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/v3dang/vex/internal/coordinator"
	"github.com/v3dang/vex/internal/ring"
	"github.com/v3dang/vex/internal/store"
)

func setupTestServer(t *testing.T) *Server {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	r := ring.New(32)
	r.AddNode(ring.Node{ID: "node-1", GrpcAddr: "127.0.0.1:7070", HttpAddr: "127.0.0.1:8080"})

	c := coordinator.New(coordinator.Config{
		NodeID:      "node-1",
		Replicas:    1,
		WriteQuorum: 1,
		Timeout:     time.Second,
	}, s, r)
	t.Cleanup(func() { c.Close() })

	return NewServer("node-1", s, r, c)
}

func TestHTTPInsertAndSearch(t *testing.T) {
	srv := setupTestServer(t)

	// 1. Insert vector
	body := map[string]any{
		"id":      "doc-123",
		"vector":  []float32{0.5, 0.5, 0.0},
		"payload": map[string]string{"title": "Distributed Systems in Go"},
	}
	raw, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/vectors", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Get vector
	getReq := httptest.NewRequest(http.MethodGet, "/vectors?id=doc-123", nil)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var getResp struct {
		Found   bool      `json:"found"`
		ID      string    `json:"id"`
		Vector  []float32 `json:"vector"`
		Payload map[string]string `json:"payload"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !getResp.Found || getResp.ID != "doc-123" || getResp.Payload["title"] != "Distributed Systems in Go" {
		t.Errorf("unexpected get response: %+v", getResp)
	}

	// 3. Search vector
	searchBody := map[string]any{
		"vector": []float32{0.5, 0.5, 0.0},
		"k":      5,
	}
	rawSearch, _ := json.Marshal(searchBody)
	searchReq := httptest.NewRequest(http.MethodPost, "/vectors/search", bytes.NewReader(rawSearch))
	searchRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(searchRec, searchReq)

	if searchRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on search, got %d: %s", searchRec.Code, searchRec.Body.String())
	}

	var searchResp struct {
		Count   int `json:"count"`
		Results []struct {
			ID       string  `json:"id"`
			Distance float32 `json:"distance"`
		} `json:"results"`
	}
	if err := json.Unmarshal(searchRec.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("unmarshal search error: %v", err)
	}
	if searchResp.Count != 1 || searchResp.Results[0].ID != "doc-123" {
		t.Fatalf("expected 1 result with doc-123, got %+v", searchResp)
	}
	if searchResp.Results[0].Distance > 0.001 {
		t.Errorf("expected near 0 distance, got %f", searchResp.Results[0].Distance)
	}
}

func TestHTTPHealthAndCluster(t *testing.T) {
	srv := setupTestServer(t)

	// Health
	hReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	hRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(hRec, hReq)
	if hRec.Code != http.StatusOK {
		t.Fatalf("health returned %d", hRec.Code)
	}

	// Cluster
	cReq := httptest.NewRequest(http.MethodGet, "/cluster", nil)
	cRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(cRec, cReq)
	if cRec.Code != http.StatusOK {
		t.Fatalf("cluster returned %d", cRec.Code)
	}
}
