package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/signwell/internal/httpapi"
	"github.com/example/signwell/internal/store"
)

func TestHealthAndJobFlow(t *testing.T) {
	mem := store.NewMemoryStore()
	mem.Clear()
	handler := httpapi.NewServer(mem)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status %d", rec.Code)
	}

	payload, _ := json.Marshal(map[string]any{
		"code":      "sw-400",
		"client":    "Harbor Inn",
		"title":     "Porch hanging sign",
		"substrate": "wood",
		"medium":    "enamel",
		"dueOn":     "2026-09-05",
		"tags":      []string{"inn"},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d body %s", rec.Code, rec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}
	if created["code"] != "SW-400" {
		t.Fatalf("expected SW-400, got %v", created["code"])
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+id+"/advance", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("advance status %d body %s", rec.Code, rec.Body.String())
	}

	notePayload, _ := json.Marshal(map[string]any{
		"message": "Sketch approved.",
		"status":  "lettered",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+id+"/notes", bytes.NewReader(notePayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("note status %d body %s", rec.Code, rec.Body.String())
	}

	var job map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &job)
	if job["status"] != "lettered" {
		t.Fatalf("expected lettered, got %v", job["status"])
	}
}

func TestNotFound(t *testing.T) {
	mem := store.NewMemoryStore()
	mem.Clear()
	handler := httpapi.NewServer(mem)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing-id", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
