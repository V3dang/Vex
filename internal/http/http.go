package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/v3dang/vex/internal/coordinator"
	"github.com/v3dang/vex/internal/ring"
	"github.com/v3dang/vex/internal/store"
	pb "github.com/v3dang/vex/pb"
)

type Server struct {
	nodeID      string
	store       *store.Store
	ring        *ring.Ring
	coordinator *coordinator.Coordinator
	mux         *http.ServeMux
}

func NewServer(nodeID string, s *store.Store, r *ring.Ring, c *coordinator.Coordinator) *Server {
	srv := &Server{
		nodeID:      nodeID,
		store:       s,
		ring:        r,
		coordinator: c,
		mux:         http.NewServeMux(),
	}
	srv.routes()
	return srv
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/vectors", s.handleVectors)
	s.mux.HandleFunc("/vectors/search", s.handleSearch)
	s.mux.HandleFunc("/cluster", s.handleCluster)
	s.mux.HandleFunc("/health", s.handleHealth)
}

type insertPayload struct {
	ID      string          `json:"id"`
	Vector  []float32       `json:"vector"`
	Payload json.RawMessage `json:"payload"`
}

type searchPayload struct {
	Vector []float32 `json:"vector"`
	K      int       `json:"k"`
	Ef     int       `json:"ef"`
}

func (s *Server) handleVectors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		var p insertPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, `{"error": "invalid json body"}`, http.StatusBadRequest)
			return
		}
		if p.ID == "" || len(p.Vector) == 0 {
			http.Error(w, `{"error": "id and vector are required"}`, http.StatusBadRequest)
			return
		}

		err := s.coordinator.CoordinatedInsert(r.Context(), &pb.InsertRequest{
			Id:      p.ID,
			Vector:  p.Vector,
			Payload: []byte(p.Payload),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error(), "success": false})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "id": p.ID})

	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id == "" {
			id = strings.TrimPrefix(r.URL.Path, "/vectors/")
		}
		if id == "" || id == "/vectors" {
			http.Error(w, `{"error": "id parameter is required"}`, http.StatusBadRequest)
			return
		}

		resp, err := s.coordinator.CoordinatedGet(r.Context(), &pb.GetRequest{Id: id})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		if !resp.Found {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"found": false, "id": id})
			return
		}

		var rawPayload any = string(resp.Payload)
		var jsonVal any
		if json.Unmarshal(resp.Payload, &jsonVal) == nil {
			rawPayload = jsonVal
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"found":   true,
			"id":      resp.Id,
			"vector":  resp.Vector,
			"payload": rawPayload,
		})

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			id = strings.TrimPrefix(r.URL.Path, "/vectors/")
		}
		if id == "" {
			http.Error(w, `{"error": "id parameter is required"}`, http.StatusBadRequest)
			return
		}

		err := s.coordinator.CoordinatedDelete(r.Context(), &pb.DeleteRequest{Id: id})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error(), "success": false})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "id": id})

	default:
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var p searchPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, `{"error": "invalid json body"}`, http.StatusBadRequest)
		return
	}
	if len(p.Vector) == 0 {
		http.Error(w, `{"error": "vector is required"}`, http.StatusBadRequest)
		return
	}
	if p.K <= 0 {
		p.K = 10
	}

	results, err := s.coordinator.CoordinatedSearch(r.Context(), &pb.SearchRequest{
		Vector: p.Vector,
		K:      int32(p.K),
		Ef:     int32(p.Ef),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}

	type jsonResult struct {
		ID       string `json:"id"`
		Distance float32 `json:"distance"`
		Payload  any    `json:"payload,omitempty"`
	}

	out := make([]jsonResult, len(results))
	for i, res := range results {
		var payload any = string(res.Payload)
		var jsonVal any
		if json.Unmarshal(res.Payload, &jsonVal) == nil {
			payload = jsonVal
		}
		out[i] = jsonResult{
			ID:       res.Id,
			Distance: res.Distance,
			Payload:  payload,
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"results": out,
		"count":   len(out),
	})
}

func (s *Server) handleCluster(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodes := s.ring.GetAllNodes()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"local_node_id": s.nodeID,
		"cluster_size":  len(nodes),
		"nodes":         nodes,
		"local_vectors": s.store.Count(),
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":        "ok",
		"node_id":       s.nodeID,
		"cluster_nodes": s.ring.NodeCount(),
		"local_vectors": s.store.Count(),
	})
}
