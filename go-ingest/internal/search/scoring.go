package search

import (
	"github.com/zanmajeric/reporadar-go-ingest/config"
)

type Confidence string

const (
	ConfidenceStrong Confidence = "strong"
	ConfidenceWeak   Confidence = "weak"
)

type Result struct {
	ID         string     `json:"id"`
	Repo       string     `json:"repo"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Similarity float64    `json:"similarity"`
	Confidence Confidence `json:"confidence"`
}

func ScoreAndRank(issues []IssueRow, cfg config.AppConfig) []Result {
	var strong []Result
	var weak []Result
	for _, issue := range issues {
		// cosine similarity in [-1, 1] from cosine distance
		sim := 1.0 - issue.Distance
		res := Result{
			ID:         issue.ID,
			Repo:       issue.Repo,
			Title:      issue.Title,
			Body:       issue.Body,
			Similarity: sim,
		}
		if sim >= cfg.StrongSimThr {
			res.Confidence = ConfidenceStrong
			strong = append(strong, res)
		} else if sim >= cfg.WeakSimThr {
			res.Confidence = ConfidenceWeak
			weak = append(weak, res)
		}
	}
	out := append([]Result{}, strong...)
	out = append(out, weak...)
	return out
}
