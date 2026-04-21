package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer() *http.ServeMux {
	s := &Store{}
	return newMux(s)
}

func TestHealth(t *testing.T) {
	mux := setupTestServer()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", resp["status"])
	}
	if resp["service"] != "aggregator" {
		t.Fatalf("expected service aggregator, got %s", resp["service"])
	}
}

func TestIngestAndAggregate(t *testing.T) {
	mux := setupTestServer()

	metrics := []Metric{
		{Name: "cpu", Value: 10.0},
		{Name: "cpu", Value: 20.0},
		{Name: "cpu", Value: 30.0},
		{Name: "mem", Value: 512.0},
	}

	for _, m := range metrics {
		body, _ := json.Marshal(m)
		req := httptest.NewRequest("POST", "/ingest", bytes.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/aggregate?name=cpu", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result AggregateResult
	json.NewDecoder(w.Body).Decode(&result)

	if result.Count != 3 {
		t.Fatalf("expected count 3, got %d", result.Count)
	}
	if result.Sum != 60.0 {
		t.Fatalf("expected sum 60, got %f", result.Sum)
	}
	if result.Avg != 20.0 {
		t.Fatalf("expected avg 20, got %f", result.Avg)
	}
	if result.Min != 10.0 {
		t.Fatalf("expected min 10, got %f", result.Min)
	}
	if result.Max != 30.0 {
		t.Fatalf("expected max 30, got %f", result.Max)
	}
}

func TestAggregateAll(t *testing.T) {
	mux := setupTestServer()

	metrics := []Metric{
		{Name: "cpu", Value: 50.0},
		{Name: "mem", Value: 256.0},
	}
	for _, m := range metrics {
		body, _ := json.Marshal(m)
		req := httptest.NewRequest("POST", "/ingest", bytes.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}

	req := httptest.NewRequest("GET", "/aggregate", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var results []AggregateResult
	json.NewDecoder(w.Body).Decode(&results)
	if len(results) != 2 {
		t.Fatalf("expected 2 aggregates, got %d", len(results))
	}
}

func TestAggregateNotFound(t *testing.T) {
	mux := setupTestServer()
	req := httptest.NewRequest("GET", "/aggregate?name=nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestIngestInvalidBody(t *testing.T) {
	mux := setupTestServer()
	req := httptest.NewRequest("POST", "/ingest", bytes.NewReader([]byte("not-json")))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestIngestEmptyName(t *testing.T) {
	mux := setupTestServer()
	body, _ := json.Marshal(Metric{Name: "", Value: 1.0})
	req := httptest.NewRequest("POST", "/ingest", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestIngestMethodNotAllowed(t *testing.T) {
	mux := setupTestServer()
	req := httptest.NewRequest("GET", "/ingest", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
