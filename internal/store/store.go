package store

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"sync"

	"github.com/coder/hnsw"
	"github.com/v3dang/vex/internal/wal"
)

// Node holds the vector + payload together.
// The hnsw library stores vectors; we store metadata separately.
type Node struct {
	ID      string
	Vector  []float32
	Payload []byte
	Deleted bool
}

type Store struct {
	index   *hnsw.Graph[string]
	meta    map[string]*Node // ID -> full node with payload
	wal     *wal.WAL
	mu      sync.RWMutex
	dataDir string
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	w, err := wal.Open(filepath.Join(dataDir, "vex.wal"))
	if err != nil {
		return nil, err
	}

	s := &Store{
		index:   hnsw.NewGraph[string](),
		meta:    make(map[string]*Node),
		wal:     w,
		dataDir: dataDir,
	}

	// Try loading persisted index snapshot first
	s.loadSnapshot()

	// Replay WAL on top of snapshot (catches anything after last snapshot)
	records, err := w.Replay()
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		switch r.Op {
		case wal.OpInsert:
			node := &Node{ID: r.ID, Vector: r.Vector, Payload: r.Payload}
			s.index.Add(hnsw.MakeNode(r.ID, r.Vector))
			s.meta[r.ID] = node
		case wal.OpDelete:
			s.index.Delete(r.ID)
			if n, ok := s.meta[r.ID]; ok {
				n.Deleted = true
			}
		}
	}

	return s, nil
}

func (s *Store) Insert(id string, vector []float32, payload []byte) error {
	// WAL first
	if err := s.wal.Write(wal.Record{
		Op: wal.OpInsert, ID: id, Vector: vector, Payload: payload,
	}); err != nil {
		return err
	}

	node := &Node{ID: id, Vector: vector, Payload: payload}

	s.mu.Lock()
	s.index.Add(hnsw.MakeNode(id, vector))
	s.meta[id] = node
	s.mu.Unlock()

	return nil
}

type Result struct {
	ID       string
	Distance float32
	Payload  []byte
}

func (s *Store) Search(query []float32, k int) ([]Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// coder/hnsw Search returns []hnsw.Node[string]
	neighbors := s.index.Search(query, k)

	distFn := s.index.Distance
	if distFn == nil {
		distFn = hnsw.CosineDistance
	}

	out := make([]Result, 0, len(neighbors))
	for _, nb := range neighbors {
		nID := nb.Key
		meta := s.meta[nID]
		if meta != nil && meta.Deleted {
			continue
		}
		payload := []byte{}
		vec := nb.Value
		if meta != nil {
			payload = meta.Payload
			if len(vec) == 0 {
				vec = meta.Vector
			}
		}
		dist := distFn(query, vec)
		out = append(out, Result{
			ID:       nID,
			Distance: dist,
			Payload:  payload,
		})
	}
	return out, nil
}

func (s *Store) Get(id string) (*Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.meta[id]
	if !ok || n.Deleted {
		return nil, false
	}
	return n, true
}

func (s *Store) Delete(id string) error {
	if err := s.wal.Write(wal.Record{Op: wal.OpDelete, ID: id}); err != nil {
		return err
	}

	s.mu.Lock()
	s.index.Delete(id)
	if n, ok := s.meta[id]; ok {
		n.Deleted = true
	}
	s.mu.Unlock()

	return nil
}

func (s *Store) Count() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.meta))
}

// Snapshot serializes the index to disk periodically.
// This keeps WAL from growing forever — after a snapshot,
// old WAL entries can be truncated.
func (s *Store) Snapshot() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := os.Create(filepath.Join(s.dataDir, "vex.snapshot"))
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(s.meta)
}

func (s *Store) loadSnapshot() {
	f, err := os.Open(filepath.Join(s.dataDir, "vex.snapshot"))
	if err != nil {
		return // no snapshot yet, that's fine
	}
	defer f.Close()

	var meta map[string]*Node
	if err := gob.NewDecoder(f).Decode(&meta); err != nil {
		return
	}

	for _, node := range meta {
		if !node.Deleted {
			s.index.Add(hnsw.MakeNode(node.ID, node.Vector))
		}
		s.meta[node.ID] = node
	}
}

func (s *Store) Close() error {
	s.Snapshot() // best-effort snapshot on clean shutdown
	return s.wal.Close()
}
