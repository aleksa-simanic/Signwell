package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/example/signwell/internal/models"
	"github.com/example/signwell/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	store *store.MemoryStore
}

func NewServer(mem *store.MemoryStore) http.Handler {
	s := &Server{store: mem}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors)

	r.Get("/", s.root)
	r.Get("/health", s.health)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/jobs", s.listJobs)
		api.Post("/jobs", s.createJob)
		api.Get("/jobs/{jobID}", s.getJob)
		api.Patch("/jobs/{jobID}", s.updateJob)
		api.Delete("/jobs/{jobID}", s.deleteJob)
		api.Post("/jobs/{jobID}/advance", s.advanceJob)
		api.Post("/jobs/{jobID}/notes", s.addNote)
	})

	return r
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) root(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"service": "Signwell",
		"version": "1.0.0",
		"health":  "/health",
		"jobs":    "/api/v1/jobs",
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "Signwell",
	})
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	status := models.JobStatus(r.URL.Query().Get("status"))
	substrate := models.Substrate(r.URL.Query().Get("substrate"))
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	writeJSON(w, http.StatusOK, s.store.ListJobs(status, substrate, q, limit, offset))
}

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var req models.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	job, err := s.store.CreateJob(req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.store.GetJob(chi.URLParam(r, "jobID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) updateJob(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	job, err := s.store.UpdateJob(chi.URLParam(r, "jobID"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) deleteJob(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteJob(chi.URLParam(r, "jobID")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) advanceJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.store.Advance(chi.URLParam(r, "jobID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) addNote(w http.ResponseWriter, r *http.Request) {
	var req models.AddNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	job, err := s.store.AddNote(chi.URLParam(r, "jobID"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrBadRequest):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]any{
		"title":  http.StatusText(status),
		"detail": detail,
		"status": status,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
