package search

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

func ScoreAndRank(issues []IssueRow, limit int, thresholds Thresholds) []Result {
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
