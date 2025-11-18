package api_server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmajeric/reporadar-go-ingest/embedder"
	"github.com/zanmajeric/reporadar-go-ingest/utils"
)

type Issue struct {
	ID        string    `json:"id"`
	Repo      string    `json:"repo"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Labels    []string  `json:"labels"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type Server struct {
	http     http.Server
	router   *http.ServeMux
	db       *pgxpool.Pool
	embedder *embedder.Client
}

func NewServer(port int, db *pgxpool.Pool, embedder *embedder.Client) *Server {
	s := Server{
		db:       db,
		router:   http.NewServeMux(),
		embedder: embedder,
	}
	s.routes()
	s.http = http.Server{
		Addr:    fmt.Sprintf(":%d", port),
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

func (s *Server) routes() {
	s.router.HandleFunc("GET /health", s.handleHealth)
	s.router.HandleFunc("GET /repos", s.handleRepo)
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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusBadRequest)
	}

	type Req struct {
		Repo string `json:"repo"`
	}

	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "json error: "+err.Error(), http.StatusBadRequest)
	}
	if req.Repo == "" {
		http.Error(w, "repo required", http.StatusBadRequest)
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

	b, err := os.ReadFile("../data/mock_issues.json")
	if err != nil {
		http.Error(w, "mock file not found: "+err.Error(), http.StatusBadRequest)
	}

	var issues []Issue
	if err := json.Unmarshal(b, &issues); err != nil {
		http.Error(w, "invalid json structure: "+err.Error(), http.StatusBadRequest)
	}

	//---- STORE INTO POSTGRES ----
	ctx := r.Context()

	for _, iss := range issues {
		_, err := s.db.Exec(ctx,
			`INSERT INTO issues (id, repo, title, body, labels, created_at, updated_at)
	        VALUES ($1,$2,$3,$4,$5,$6,$7)
	        ON CONFLICT (id) DO NOTHING`,
			iss.ID, iss.Repo, iss.Title, iss.Body, iss.Labels, iss.CreatedAt, iss.UpdatedAt,
		)
		if err != nil {
			http.Error(w, "Db error: "+err.Error(), http.StatusInternalServerError)
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

	var out []Issue
	for rows.Next() {
		var iss Issue
		if err := rows.Scan(&iss.ID, &iss.Title, &iss.Body, &iss.Labels, &iss.CreatedAt); err != nil {
			http.Error(w, "Db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, iss)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)

}

type Confidence string

const (
	ConfidenceStrong Confidence = "strong"
	ConfidenceWeak   Confidence = "weak"
)

type searchResult struct {
	ID         string     `json:"id"`
	Repo       string     `json:"repo"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Similarity float64    `json:"similarity"`
	Confidence Confidence `json:"confidence"`
}

type searchResponse struct {
	Results      []searchResult `json:"results"`
	Message      string         `json:"message"`
	StrongSimThr float64        `json:"strong_sim_thr"`
	WeakSimThr   float64        `json:"weak_sim_thr"`
}

// TODO: move logic to service layer
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

	embedderStartTime := time.Now()
	emb, err := s.embedder.Embed(ctx, searchQuery)
	if err != nil {
		http.Error(w, "embed call error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	embedderReqTime := time.Since(embedderStartTime)

	vectorLiteral := utils.EmbeddingToVectorLiteral(emb)

	const qSQL = `
		SELECT id, repo, title, body, embedding <=> $1::vector AS distance
		FROM issues
		WHERE repo = $2
		ORDER BY embedding <=> $1::vector
		LIMIT $3;
	`

	sqlStartTime := time.Now()
	rows, err := s.db.Query(ctx, qSQL, vectorLiteral, repo, limit)
	if err != nil {
		http.Error(w, "search query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sqlProcTime := time.Since(sqlStartTime)

	defer rows.Close()

	var strongMatches []searchResult
	var weakMatches []searchResult
	const strongSim = 0.5
	const weakSim = 0.3
	for rows.Next() {
		var (
			id       string
			repoVal  string
			title    sql.NullString
			body     sql.NullString
			distance float64
		)

		if err := rows.Scan(&id, &repoVal, &title, &body, &distance); err != nil {
			http.Error(w, "search scan failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// cosine similarity in [-1, 1] from cosine distance
		sim := 1.0 - distance
		res := searchResult{
			ID:         id,
			Repo:       repoVal,
			Title:      title.String,
			Body:       body.String,
			Similarity: sim,
		}
		if sim >= strongSim {
			res.Confidence = ConfidenceStrong
			strongMatches = append(strongMatches, res)
		} else if sim >= weakSim {
			res.Confidence = ConfidenceWeak
			weakMatches = append(weakMatches, res)
		} else {
			log.Printf(
				"filtered-out q=%q title=%q sim=%.4f (strong>=%.2f, weak>=%.2f)",
				searchQuery, title.String, sim, strongSim, weakSim,
			)
		}
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "search rows error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var results []searchResult
	switch {
	case len(strongMatches) > 0 && len(strongMatches) < limit:
		results = append(results, strongMatches...)
		remaining := limit - len(strongMatches)
		if remaining > len(weakMatches) {
			remaining = len(weakMatches)
		}
		results = append(results, weakMatches[:remaining]...)
	case len(strongMatches) > 0:
		results = strongMatches
	case len(weakMatches) > 0:
		results = weakMatches
	}

	searchRsp := searchResponse{Results: results, StrongSimThr: strongSim, WeakSimThr: weakSim}
	if len(results) == 0 {
		searchRsp.Message = "no sufficiently similar issues found"
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(searchRsp) // added respMsg

	reqTime := time.Since(start)
	log.Printf("[/search] request time: %v | embedding time: %v | sql time: %v \n ------", reqTime, embedderReqTime, sqlProcTime)

}
