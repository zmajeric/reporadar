package search

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zanmajeric/reporadar-go-ingest/config"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type IssueRepository interface {
	SearchByVector(ctx context.Context, repo string, vector []float32, limit int) ([]IssueRow, error)
}

type IssueRow struct {
	ID        string    `json:"id"`
	Repo      string    `json:"repo"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Labels    []string  `json:"labels"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	Distance  float64   //embedding
}

type Service struct {
	embedder Embedder
	repo     IssueRepository
	cfg      *config.AppConfig
}

type Thresholds struct {
	Strong float64
	Weak   float64
}

func New(embedder Embedder, issuesRep IssueRepository, cfg config.AppConfig) *Service {
	return &Service{
		embedder: embedder,
		repo:     issuesRep,
		cfg:      &cfg,
	}
}

func (s *Service) Search(ctx context.Context, repo, query string, limit int) ([]Result, error) {
	emb, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	issues, err := s.repo.SearchByVector(ctx, repo, emb, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	log.Printf("[search] repo=%s q=%q rows=%d", repo, query, len(issues))

	thresholds := Thresholds{
		Strong: s.cfg.StrongSimThr,
		Weak:   s.cfg.WeakSimThr,
	}
	return ScoreAndRank(issues, limit, thresholds), nil
}
