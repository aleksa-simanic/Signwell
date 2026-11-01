package store_test

import (
	"errors"
	"testing"

	"github.com/example/signwell/internal/models"
	"github.com/example/signwell/internal/store"
)

func sampleJob(code string) models.CreateJobRequest {
	return models.CreateJobRequest{
		Code:      code,
		Client:    "Harbor Inn",
		Title:     "Porch hanging sign",
		Substrate: models.SubstrateWood,
		Medium:    models.MediumEnamel,
		DueOn:     "2026-09-01",
	}
}

func TestCreateAndGetJob(t *testing.T) {
	s := store.NewMemoryStore()
	s.Clear()

	created, err := s.CreateJob(sampleJob("sw-210"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Code != "SW-210" {
		t.Fatalf("expected SW-210, got %s", created.Code)
	}
	if created.Status != models.StatusQuoted {
		t.Fatalf("expected quoted, got %s", created.Status)
	}

	got, err := s.GetJob(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "Porch hanging sign" {
		t.Fatalf("unexpected title: %s", got.Title)
	}
}

func TestListFiltersByStatusAndQuery(t *testing.T) {
	s := store.NewMemoryStore()
	s.Clear()

	_, _ = s.CreateJob(sampleJob("SW-A"))
	second := sampleJob("SW-B")
	second.Title = "Window goldleaf"
	second.Client = "Civic Arts Week"
	second.Substrate = models.SubstrateGlass
	second.Medium = models.MediumGoldleaf
	second.Status = models.StatusQuoted
	_, _ = s.CreateJob(second)

	listed := s.ListJobs(models.StatusQuoted, models.SubstrateGlass, "goldleaf", 50, 0)
	if listed.Total != 1 {
		t.Fatalf("expected 1 result, got %d", listed.Total)
	}
	if listed.Items[0].Code != "SW-B" {
		t.Fatalf("unexpected code: %s", listed.Items[0].Code)
	}
}

func TestAdvanceAndNotes(t *testing.T) {
	s := store.NewMemoryStore()
	s.Clear()

	job, err := s.CreateJob(sampleJob("SW-C"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	advanced, err := s.Advance(job.ID)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if advanced.Status != models.StatusSketched {
		t.Fatalf("expected sketched, got %s", advanced.Status)
	}

	noted, err := s.AddNote(job.ID, models.AddNoteRequest{
		Message: "Layout approved on tracing paper.",
		Status:  models.StatusLettered,
	})
	if err != nil {
		t.Fatalf("note: %v", err)
	}
	if noted.Status != models.StatusLettered {
		t.Fatalf("expected lettered, got %s", noted.Status)
	}
	if len(noted.Notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(noted.Notes))
	}
}

func TestDuplicateCodeConflict(t *testing.T) {
	s := store.NewMemoryStore()
	s.Clear()

	_, err := s.CreateJob(sampleJob("SW-DUP"))
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err = s.CreateJob(sampleJob("sw-dup"))
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestValidationErrors(t *testing.T) {
	s := store.NewMemoryStore()
	s.Clear()

	_, err := s.CreateJob(models.CreateJobRequest{Code: "  "})
	if !errors.Is(err, store.ErrBadRequest) {
		t.Fatalf("expected bad request, got %v", err)
	}
}
