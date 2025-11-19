package search

import "log"

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

// ScoreAndRank For normalized vectors, distance = -dot(u, v), so we define similarity = -distance ∈ [-1, 1].
func ScoreAndRank(issues []IssueRow, limit int, thresholds Thresholds) []Result {
	var strong []Result
	var weak []Result
	for _, issue := range issues {
		sim := -issue.Distance
		res := Result{
			ID:         issue.ID,
			Repo:       issue.Repo,
			Title:      issue.Title,
			Body:       issue.Body,
			Similarity: sim,
		}
		log.Printf("issue: [ %v ] \n distance: %v | similarity: %v", res.Title, issue.Distance, res.Similarity)
		switch {
		case sim >= thresholds.Strong:
			res.Confidence = ConfidenceStrong
			strong = append(strong, res)
		case sim >= thresholds.Weak:
			res.Confidence = ConfidenceWeak
			weak = append(weak, res)
		}
	}

	out := make([]Result, 0, len(issues))
	for _, res := range strong {
		if len(out) >= limit {
			return out
		}
		out = append(out, res)
	}
	for _, res := range weak {
		if len(out) >= limit {
			return out
		}
		out = append(out, res)
	}
	if len(out) > limit {
		out = out[:limit]
	}

	return out
}
