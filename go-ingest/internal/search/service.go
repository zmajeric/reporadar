package search

import (
	"context"
	"errors"
	"time"

	"github.com/zanmajeric/reporadar-go-ingest/config"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type IssueRepository interface {
	SearchByVector(ctx context.Context, repo, query string, vector []float32, limit int) ([]IssueRow, error)
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
	ctx      context.Context
	cfg      *config.AppConfig
}

func New(ctx context.Context, embedder Embedder, issuesRep IssueRepository, cfg config.AppConfig) *Service {
	return &Service{
		embedder: embedder,
		repo:     issuesRep,
		ctx:      ctx,
		cfg:      &cfg,
	}
}

func (s *Service) Search(ctx context.Context, repo, query string, limit int) ([]Result, error) {
	emb, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, errors.New("embedding failed")
	}

	issues, err := s.repo.SearchByVector(s.ctx, repo, query, emb, limit)
	if err != nil {
		return nil, errors.New("search failed")
	}

	return ScoreAndRank(issues, *s.cfg), nil
}
