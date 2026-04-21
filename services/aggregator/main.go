package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
)

type Metric struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Tags      map[string]string `json:"tags"`
	Timestamp float64           `json:"timestamp"`
}

type AggregateResult struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Sum   float64 `json:"sum"`
	Avg   float64 `json:"avg"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

type Store struct {
	mu      sync.RWMutex
	metrics []Metric
}

var store = &Store{}

func (s *Store) Add(m Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = append(s.metrics, m)
}

func (s *Store) Aggregate(name string) *AggregateResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var values []float64
	for _, m := range s.metrics {
		if m.Name == name {
			values = append(values, m.Value)
		}
	}
	if len(values) == 0 {
		return nil
	}

	result := &AggregateResult{
		Name:  name,
		Count: len(values),
		Min:   math.MaxFloat64,
		Max:   -math.MaxFloat64,
	}
	for _, v := range values {
		result.Sum += v
		if v < result.Min {
			result.Min = v
		}
		if v > result.Max {
			result.Max = v
		}
	}
	result.Avg = result.Sum / float64(result.Count)
	return result
}

func (s *Store) AllNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]bool{}
	var names []string
	for _, m := range s.metrics {
		if !seen[m.Name] {
			seen[m.Name] = true
			names = append(names, m.Name)
		}
	}
	return names
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "aggregator",
	})
}

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var m Metric
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if m.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusUnprocessableEntity)
		return
	}
	store.Add(m)
	log.Printf("Stored metric: %s = %f", m.Name, m.Value)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "stored"})
}

func aggregateHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	w.Header().Set("Content-Type", "application/json")

	if name == "" {
		names := store.AllNames()
		var results []AggregateResult
		for _, n := range names {
			if agg := store.Aggregate(n); agg != nil {
				results = append(results, *agg)
			}
		}
		json.NewEncoder(w).Encode(results)
		return
	}

	result := store.Aggregate(name)
	if result == nil {
		http.Error(w, `{"error":"no metrics found for name"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(result)
}

func newMux(s *Store) *http.ServeMux {
	store = s
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ingest", ingestHandler)
	mux.HandleFunc("/aggregate", aggregateHandler)
	return mux
}

func main() {
	port := os.Getenv("AGGREGATOR_PORT")
	if port == "" {
		port = "8002"
	}
	mux := newMux(store)
	log.Printf("Starting aggregator on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
