package store

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestStoreInsertAndSearch(t *testing.T) {
	// 1. Create a temporary directory for testing
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	// 2. Insert a vector
	vector := []float32{0.1, 0.2, 0.3}
	err = s.Insert("doc1", vector, []byte("my metadata"))
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// 3. Search for it
	results, err := s.Search(vector, 1)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) == 0 || results[0].ID != "doc1" {
		t.Errorf("expected to find doc1, got %v", results)
	}

	// Verify distance is approximately 0 for identical vector
	if results[0].Distance > 0.001 {
		t.Errorf("expected distance near 0 for identical vector, got %f", results[0].Distance)
	}

	// 4. Test Get
	node, ok := s.Get("doc1")
	if !ok || node.ID != "doc1" {
		t.Fatalf("expected to get doc1, got %v, ok=%v", node, ok)
	}
	if string(node.Payload) != "my metadata" {
		t.Errorf("expected payload 'my metadata', got %s", string(node.Payload))
	}
}

func TestStoreDistanceDifferences(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	// Insert orthogonal / distinct vectors
	_ = s.Insert("v1", []float32{1.0, 0.0, 0.0}, nil)
	_ = s.Insert("v2", []float32{0.0, 1.0, 0.0}, nil)

	// Query closest to v1
	res, err := s.Search([]float32{1.0, 0.0, 0.0}, 2)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(res) < 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].ID != "v1" {
		t.Errorf("expected v1 to be closest, got %s", res[0].ID)
	}
	// Distance to v1 should be near 0
	if res[0].Distance > 0.001 {
		t.Errorf("expected v1 distance near 0, got %f", res[0].Distance)
	}
	// Distance to orthogonal v2 should be near 1.0
	if res[1].Distance < 0.9 || res[1].Distance > 1.1 {
		t.Errorf("expected orthogonal distance near 1.0, got %f", res[1].Distance)
	}
}

func generateRandomVector(dim int) []float32 {
	vec := make([]float32, dim)
	for i := 0; i < dim; i++ {
		vec[i] = rand.Float32()
	}
	return vec
}

func BenchmarkStoreInsert(b *testing.B) {
	s, err := Open(b.TempDir())
	if err != nil {
		b.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	dim := 128 // Assuming 128-dimensional vectors for the benchmark
	vectors := make([][]float32, b.N)
	for i := 0; i < b.N; i++ {
		vectors[i] = generateRandomVector(dim)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := s.Insert(fmt.Sprintf("doc%d", i), vectors[i], nil)
		if err != nil {
			b.Fatalf("insert failed: %v", err)
		}
	}
}

func BenchmarkStoreSearch(b *testing.B) {
	s, err := Open(b.TempDir())
	if err != nil {
		b.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	dim := 128
	numDocs := 1000 // Number of documents to insert before searching
	
	for i := 0; i < numDocs; i++ {
		err := s.Insert(fmt.Sprintf("doc%d", i), generateRandomVector(dim), nil)
		if err != nil {
			b.Fatalf("insert failed: %v", err)
		}
	}

	query := generateRandomVector(dim)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.Search(query, 10)
		if err != nil {
			b.Fatalf("search failed: %v", err)
		}
	}
}
