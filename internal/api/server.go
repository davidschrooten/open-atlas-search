package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/david/open-atlas-search/config"
	"github.com/david/open-atlas-search/internal/indexer"
	"github.com/david/open-atlas-search/internal/search"
)

// Server represents the API server
type Server struct {
	searchEngine   search.SearchEngine
	indexerService *indexer.Service
	config         *config.Config
}

// NewServer creates a new API server
func NewServer(searchEngine search.SearchEngine, indexerService *indexer.Service, cfg *config.Config) *Server {
	return &Server{
		searchEngine:   searchEngine,
		indexerService: indexerService,
		config:         cfg,
	}
}

// Router setups the API routes
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/indexes/{index}/search", s.handleSearch)
	r.Get("/indexes/{index}/status", s.handleStatus)
	r.Get("/indexes", s.handleListIndexes)
	r.Get("/health", s.handleHealth)
	r.Get("/ready", s.handleReady)

	return r
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	index := chi.URLParam(r, "index")
	if index == "" {
		http.Error(w, "index parameter is required", http.StatusBadRequest)
		return
	}

	var searchReq search.SearchRequest
	// For GET requests, we might not have a body, so handle that case
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
	}

	// Set index from URL parameter
	searchReq.Index = index

	// Set defaults for empty search (Elasticsearch-like behavior)
	if searchReq.Query == nil {
		searchReq.Query = map[string]interface{}{"match_all": map[string]interface{}{}}
	}

	// Set default pagination
	if searchReq.Size == 0 {
		searchReq.Size = 100 // Default page size
	}
	// Limit maximum size to 10,000
	if searchReq.Size > 10000 {
		searchReq.Size = 10000
	}

	// Ensure From is not negative
	if searchReq.From < 0 {
		searchReq.From = 0
	}

	// Limit total results (From + Size) to 10,000
	if searchReq.From+searchReq.Size > 10000 {
		searchReq.Size = 10000 - searchReq.From
		if searchReq.Size <= 0 {
			http.Error(w, "pagination exceeds maximum limit of 10,000 documents", http.StatusBadRequest)
			return
		}
	}

	searchResult, err := s.searchEngine.Search(searchReq)
	if err != nil {
		log.Printf("Search error: %v", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	response(w, http.StatusOK, searchResult)
}

func (s *Server) handleListIndexes(w http.ResponseWriter, r *http.Request) {
	indexes, err := s.searchEngine.ListIndexes()
	if err != nil {
		log.Printf("List indexes error: %v", err)
		http.Error(w, "failed to list indexes", http.StatusInternalServerError)
		return
	}

	response(w, http.StatusOK, map[string]interface{}{
		"indexes": indexes,
		"total":   len(indexes),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	index := chi.URLParam(r, "index")
	if index == "" {
		http.Error(w, "index parameter is required", http.StatusBadRequest)
		return
	}

	indexes, err := s.searchEngine.ListIndexes()
	if err != nil {
		log.Printf("Status error: %v", err)
		http.Error(w, "failed to get status", http.StatusInternalServerError)
		return
	}

	// Find the specific index
	var targetIndex *search.IndexInfo
	for i, idx := range indexes {
		if idx.Name == index {
			targetIndex = &indexes[i]
			break
		}
	}

	if targetIndex == nil {
		http.Error(w, "index not found", http.StatusNotFound)
		return
	}

	// Create status response for the specific index
	status := map[string]interface{}{
		"service": "open-atlas-search",
		"status":  "running",
		"index":   *targetIndex,
	}

	response(w, http.StatusOK, status)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Simple health check
	response(w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if search engine is initialized
	if s.searchEngine == nil {
		http.Error(w, "search engine not initialized", http.StatusServiceUnavailable)
		return
	}

	// Check if indexer service is initialized  
	if s.indexerService == nil {
		http.Error(w, "indexer service not initialized", http.StatusServiceUnavailable)
		return
	}

	// Create a simple readiness check by trying to list indexes
	// This will verify that the search engine is working
	if _, err := s.searchEngine.ListIndexes(); err != nil {
		log.Printf("Readiness check failed - cannot list indexes: %v", err)
		http.Error(w, "search engine not ready", http.StatusServiceUnavailable)
		return
	}

	// If we have any configured indexes, verify at least one exists
	if len(s.config.Indexes) > 0 {
		indexes, err := s.searchEngine.ListIndexes()
		if err != nil || len(indexes) == 0 {
			log.Printf("Readiness check failed - no indexes available")
			http.Error(w, "no indexes available", http.StatusServiceUnavailable)
			return
		}
	}

	response(w, http.StatusOK, map[string]interface{}{
		"status": "ready",
		"checks": map[string]string{
			"searchEngine": "ok",
			"indexerService": "ok",
			"indexes": "ok",
		},
	})
}

func response(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Unable to encode response: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
