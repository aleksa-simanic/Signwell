package store

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/example/signwell/internal/models"
	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrBadRequest = errors.New("bad request")
	ErrConflict   = errors.New("conflict")
)

type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]models.Job
}

func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{jobs: make(map[string]models.Job)}
	s.seed()
	return s
}

func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = make(map[string]models.Job)
}

func (s *MemoryStore) CreateJob(req models.CreateJobRequest) (models.Job, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	client := strings.TrimSpace(req.Client)
	title := strings.TrimSpace(req.Title)
	if err := validateJobFields(code, client, title, req.Substrate, req.Medium, req.DueOn); err != nil {
		return models.Job{}, err
	}

	status := req.Status
	if status == "" {
		status = models.StatusQuoted
	}
	if !validStatus(status) {
		return models.Job{}, fmt.Errorf("%w: invalid status", ErrBadRequest)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUniqueCode(code, ""); err != nil {
		return models.Job{}, err
	}

	now := time.Now().UTC()
	job := models.Job{
		ID:        uuid.NewString(),
		Code:      code,
		Client:    client,
		Title:     title,
		Substrate: req.Substrate,
		Medium:    req.Medium,
		DueOn:     req.DueOn,
		Status:    status,
		Notes:     []models.Note{},
		Tags:      normalizeTags(req.Tags),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.jobs[job.ID] = job
	return job, nil
}

func (s *MemoryStore) GetJob(id string) (models.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return models.Job{}, fmt.Errorf("%w: job %s not found", ErrNotFound, id)
	}
	return job, nil
}

func (s *MemoryStore) ListJobs(status models.JobStatus, substrate models.Substrate, q string, limit, offset int) models.JobListResponse {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]models.Job, 0, len(s.jobs))
	needle := strings.ToLower(strings.TrimSpace(q))
	for _, job := range s.jobs {
		if status != "" && job.Status != status {
			continue
		}
		if substrate != "" && job.Substrate != substrate {
			continue
		}
		if needle != "" {
			hay := strings.ToLower(job.Code + " " + job.Client + " " + job.Title)
			if !strings.Contains(hay, needle) {
				continue
			}
		}
		items = append(items, job)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].DueOn == items[j].DueOn {
			return items[i].Code < items[j].Code
		}
		return items[i].DueOn < items[j].DueOn
	})

	total := len(items)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return models.JobListResponse{
		Items:  items[offset:end],
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

func (s *MemoryStore) UpdateJob(id string, req models.UpdateJobRequest) (models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return models.Job{}, fmt.Errorf("%w: job %s not found", ErrNotFound, id)
	}

	if req.Code != nil {
		code := strings.ToUpper(strings.TrimSpace(*req.Code))
		if code == "" {
			return models.Job{}, fmt.Errorf("%w: code must not be blank", ErrBadRequest)
		}
		if err := s.ensureUniqueCode(code, id); err != nil {
			return models.Job{}, err
		}
		job.Code = code
	}
	if req.Client != nil {
		client := strings.TrimSpace(*req.Client)
		if client == "" {
			return models.Job{}, fmt.Errorf("%w: client must not be blank", ErrBadRequest)
		}
		job.Client = client
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return models.Job{}, fmt.Errorf("%w: title must not be blank", ErrBadRequest)
		}
		job.Title = title
	}
	if req.Substrate != nil {
		if !validSubstrate(*req.Substrate) {
			return models.Job{}, fmt.Errorf("%w: invalid substrate", ErrBadRequest)
		}
		job.Substrate = *req.Substrate
	}
	if req.Medium != nil {
		if !validMedium(*req.Medium) {
			return models.Job{}, fmt.Errorf("%w: invalid medium", ErrBadRequest)
		}
		job.Medium = *req.Medium
	}
	if req.DueOn != nil {
		if !validDueOn(*req.DueOn) {
			return models.Job{}, fmt.Errorf("%w: dueOn must be YYYY-MM-DD", ErrBadRequest)
		}
		job.DueOn = *req.DueOn
	}
	if req.Status != nil {
		if !validStatus(*req.Status) {
			return models.Job{}, fmt.Errorf("%w: invalid status", ErrBadRequest)
		}
		job.Status = *req.Status
	}
	if req.Tags != nil {
		job.Tags = normalizeTags(*req.Tags)
	}

	job.UpdatedAt = time.Now().UTC()
	s.jobs[id] = job
	return job, nil
}

func (s *MemoryStore) DeleteJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[id]; !ok {
		return fmt.Errorf("%w: job %s not found", ErrNotFound, id)
	}
	delete(s.jobs, id)
	return nil
}

func (s *MemoryStore) Advance(id string) (models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return models.Job{}, fmt.Errorf("%w: job %s not found", ErrNotFound, id)
	}
	next, ok := models.StatusFlow[job.Status]
	if !ok {
		return models.Job{}, fmt.Errorf("%w: job is already %s", ErrConflict, job.Status)
	}
	job.Status = next
	job.UpdatedAt = time.Now().UTC()
	s.jobs[id] = job
	return job, nil
}

func (s *MemoryStore) AddNote(id string, req models.AddNoteRequest) (models.Job, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return models.Job{}, fmt.Errorf("%w: message is required", ErrBadRequest)
	}
	if len(message) > 2000 {
		return models.Job{}, fmt.Errorf("%w: message must be at most 2000 characters", ErrBadRequest)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return models.Job{}, fmt.Errorf("%w: job %s not found", ErrNotFound, id)
	}

	status := req.Status
	if status == "" {
		status = job.Status
	}
	if !validStatus(status) {
		return models.Job{}, fmt.Errorf("%w: invalid status", ErrBadRequest)
	}

	now := time.Now().UTC()
	job.Notes = append(job.Notes, models.Note{
		ID:        uuid.NewString(),
		Message:   message,
		Status:    status,
		CreatedAt: now,
	})
	job.Status = status
	job.UpdatedAt = now
	s.jobs[id] = job
	return job, nil
}

func (s *MemoryStore) seed() {
	_, _ = s.CreateJob(models.CreateJobRequest{
		Code:      "SW-101",
		Client:    "Harbor Inn",
		Title:     "Porch hanging sign",
		Substrate: models.SubstrateWood,
		Medium:    models.MediumEnamel,
		DueOn:     "2026-08-22",
		Status:    models.StatusLettered,
		Tags:      []string{"inn", "rush"},
	})
	_, _ = s.CreateJob(models.CreateJobRequest{
		Code:      "SW-102",
		Client:    "Civic Arts Week",
		Title:     "Window goldleaf",
		Substrate: models.SubstrateGlass,
		Medium:    models.MediumGoldleaf,
		DueOn:     "2026-08-28",
		Status:    models.StatusQuoted,
		Tags:      []string{"event"},
	})
	_, _ = s.CreateJob(models.CreateJobRequest{
		Code:      "SW-103",
		Client:    "Nora Ellis",
		Title:     "Studio door plate",
		Substrate: models.SubstrateMetal,
		Medium:    models.MediumEnamel,
		DueOn:     "2026-08-18",
		Status:    models.StatusReady,
		Tags:      []string{"studio"},
	})
}

func (s *MemoryStore) ensureUniqueCode(code, exceptID string) error {
	for _, job := range s.jobs {
		if job.Code == code && job.ID != exceptID {
			return fmt.Errorf("%w: job code '%s' already exists", ErrConflict, code)
		}
	}
	return nil
}

func validateJobFields(code, client, title string, substrate models.Substrate, medium models.Medium, dueOn string) error {
	if code == "" {
		return fmt.Errorf("%w: code is required", ErrBadRequest)
	}
	if len(code) > 20 {
		return fmt.Errorf("%w: code must be at most 20 characters", ErrBadRequest)
	}
	if client == "" {
		return fmt.Errorf("%w: client is required", ErrBadRequest)
	}
	if len(client) > 120 {
		return fmt.Errorf("%w: client must be at most 120 characters", ErrBadRequest)
	}
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrBadRequest)
	}
	if len(title) > 160 {
		return fmt.Errorf("%w: title must be at most 160 characters", ErrBadRequest)
	}
	if !validSubstrate(substrate) {
		return fmt.Errorf("%w: invalid substrate", ErrBadRequest)
	}
	if !validMedium(medium) {
		return fmt.Errorf("%w: invalid medium", ErrBadRequest)
	}
	if !validDueOn(dueOn) {
		return fmt.Errorf("%w: dueOn must be YYYY-MM-DD", ErrBadRequest)
	}
	return nil
}

func validSubstrate(value models.Substrate) bool {
	switch value {
	case models.SubstrateWood, models.SubstrateMetal, models.SubstrateGlass, models.SubstrateVinyl:
		return true
	default:
		return false
	}
}

func validMedium(value models.Medium) bool {
	switch value {
	case models.MediumEnamel, models.MediumGoldleaf, models.MediumVinyl:
		return true
	default:
		return false
	}
}

func validStatus(value models.JobStatus) bool {
	switch value {
	case models.StatusQuoted, models.StatusSketched, models.StatusLettered,
		models.StatusFinishing, models.StatusReady, models.StatusDelivered:
		return true
	default:
		return false
	}
}

func validDueOn(value string) bool {
	if len(value) != 10 {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		item := strings.ToLower(strings.TrimSpace(tag))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
