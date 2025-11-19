package api_server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmajeric/reporadar-go-ingest/config"
	"github.com/zanmajeric/reporadar-go-ingest/internal/search"
)

type Server struct {
	http      http.Server
	router    *http.ServeMux
	db        *pgxpool.Pool
	searchSrv *search.Service
	cfg       *config.AppConfig
}

func NewServer(cfg *config.AppConfig, db *pgxpool.Pool, searchSrv *search.Service) *Server {
	s := Server{
		db:        db,
		router:    http.NewServeMux(),
		searchSrv: searchSrv,
		cfg:       cfg,
	}
	s.routes()
	s.http = http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HttpPort),
		Handler: s.router,
	}
	return &s
}

func (s *Server) Run() {
	err := s.http.ListenAndServe()
	if err != nil {
		log.Fatalf("Error starting server: %s", err)
	}
}

func (s *Server) Close(ctx context.Context) {
	err := s.http.Shutdown(ctx)
	if err != nil {
		log.Fatalf("Error shutting down http server: %s", err)
	}
}

func (s *Server) routes() {
	s.router.HandleFunc("GET /health", s.handleHealth)
	s.router.HandleFunc("POST /repos", s.handleRepo)
	s.router.HandleFunc("POST /repos/{repo}/ingest", s.handleIngest)
	s.router.HandleFunc("GET /issues", s.handleIssues)
	s.router.HandleFunc("GET /search", s.handleSearch)
	// TODO: add /repos, /repos/{id}/ingest, /issues, /search, /issues/{id}/duplicates
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := s.db.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"error","error":"%v"}`, err)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleRepo(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		Repo string `json:"repo"`
	}

	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "json error: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Repo == "" {
		http.Error(w, "repo required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","repo":"` + req.Repo + `"}`))
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	if repo == "" {
		http.Error(w, "missing repo path parameter", http.StatusBadRequest)
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode != "mock" {
		http.Error(w, "only mode=mock is supported today", http.StatusBadRequest)
		return
	}

	b, err := os.ReadFile("/Users/zmajeric/development/reporadar/data/mock_issues.json")
	if err != nil {
		http.Error(w, "mock file not found: "+err.Error(), http.StatusBadRequest)
	}

	var issues []search.IssueRow
	if err := json.Unmarshal(b, &issues); err != nil {
		http.Error(w, "invalid json structure: "+err.Error(), http.StatusBadRequest)
	}

	//---- STORE INTO POSTGRES ----
	ctx := r.Context()

	for _, iss := range issues {
		_, err := s.db.Exec(ctx,
			`
			INSERT INTO issues (id, repo, title, body, labels, created_at, updated_at)
	        VALUES ($1,$2,$3,$4,$5,$6,$7)
	        ON CONFLICT (id) DO NOTHING
	        `,
			iss.ID, iss.Repo, iss.Title, iss.Body, iss.Labels, iss.CreatedAt, iss.UpdatedAt,
		)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) handleIssues(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	if repo == "" {
		http.Error(w, "repo required", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Query(r.Context(), "SELECT id, title, body, labels, created_at FROM issues WHERE repo=$1 ORDER BY created_at", repo)
	if err != nil {
		http.Error(w, "Db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []search.IssueRow
	for rows.Next() {
		var iss search.IssueRow
		if err := rows.Scan(&iss.ID, &iss.Title, &iss.Body, &iss.Labels, &iss.CreatedAt); err != nil {
			http.Error(w, "Db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, iss)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)

}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	repo := r.URL.Query().Get("repo")
	if repo == "" {
		http.Error(w, "repo is required", http.StatusBadRequest)
		return
	}

	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		http.Error(w, "q is required", http.StatusBadRequest)
		return
	}

	limit := 10
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if l, err := strconv.Atoi(ls); err == nil && l > 0 && l <= 20 {
			limit = l
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	found, err := s.searchSrv.Search(ctx, repo, searchQuery, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := struct {
		Results      []search.Result `json:"results"`
		Message      string          `json:"message"`
		StrongSimThr float64         `json:"strong_sim_thr"`
		WeakSimThr   float64         `json:"weak_sim_thr"`
	}{
		Results:      found,
		StrongSimThr: s.cfg.StrongSimThr,
		WeakSimThr:   s.cfg.WeakSimThr,
	}
	if len(found) == 0 {
		resp.Message = "no sufficiently similar issues found"
	}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, "failed serializing response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	reqTime := time.Since(start)
	log.Printf("[/search] request time: %v", reqTime)
}
