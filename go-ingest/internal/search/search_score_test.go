package search

import "testing"

func TestScoreAndRank_StrongAndWeakClassification(t *testing.T) {
	issues := []IssueRow{
		// Remember: similarity = -distance
		// So distance -0.8 => sim 0.8 (strong)
		{
			ID:       "1",
			Repo:     "demo/reporadar",
			Title:    "Strong match",
			Body:     "something very close",
			Distance: -0.8,
		},
		// distance -0.4 => sim 0.4 (weak)
		{
			ID:       "2",
			Repo:     "demo/reporadar",
			Title:    "Weak match",
			Body:     "somewhat related",
			Distance: -0.4,
		},
		// distance -0.1 => sim 0.1 (below weak threshold, should be dropped)
		{
			ID:       "3",
			Repo:     "demo/reporadar",
			Title:    "Irrelevant",
			Body:     "totally unrelated",
			Distance: -0.1,
		},
	}

	thr := Thresholds{
		Strong: 0.6,
		Weak:   0.3,
	}

	got := ScoreAndRank(issues, 10, thr)

	if len(got) != 2 {
		t.Fatalf("expected 2 results (1 strong, 1 weak), got %d", len(got))
	}

	// Order should preserve DB order: strongs first, then weaks, both
	// in the order they appeared in `issues`.
	if got[0].ID != "1" {
		t.Errorf("expected first result ID=1 (strong), got ID=%s", got[0].ID)
	}
	if got[0].Confidence != ConfidenceStrong {
		t.Errorf("expected first result confidence=%q, got %q", ConfidenceStrong, got[0].Confidence)
	}

	if got[1].ID != "2" {
		t.Errorf("expected second result ID=2 (weak), got ID=%s", got[1].ID)
	}
	if got[1].Confidence != ConfidenceWeak {
		t.Errorf("expected second result confidence=%q, got %q", ConfidenceWeak, got[1].Confidence)
	}
}

func TestScoreAndRank_RespectsLimit(t *testing.T) {
	issues := []IssueRow{
		// 3 strong matches
		{ID: "1", Repo: "demo/reporadar", Title: "s1", Distance: -0.9}, // sim 0.9
		{ID: "2", Repo: "demo/reporadar", Title: "s2", Distance: -0.8}, // sim 0.8
		{ID: "3", Repo: "demo/reporadar", Title: "s3", Distance: -0.7}, // sim 0.7
		// 2 weak matches
		{ID: "4", Repo: "demo/reporadar", Title: "w1", Distance: -0.4},  // sim 0.4
		{ID: "5", Repo: "demo/reporadar", Title: "w2", Distance: -0.35}, // sim 0.35
	}

	thr := Thresholds{
		Strong: 0.6,
		Weak:   0.3,
	}

	limit := 2
	got := ScoreAndRank(issues, limit, thr)

	if len(got) != limit {
		t.Fatalf("expected %d results, got %d", limit, len(got))
	}

	// Since we have more strong results than the limit, we should only get strong ones.
	for i, res := range got {
		if res.Confidence != ConfidenceStrong {
			t.Errorf("result %d expected confidence=%q, got %q", i, ConfidenceStrong, res.Confidence)
		}
	}

	if got[0].ID != "1" || got[1].ID != "2" {
		t.Errorf("expected IDs [1,2], got [%s,%s]", got[0].ID, got[1].ID)
	}
}

func TestScoreAndRank_NoResultsAboveWeakThreshold(t *testing.T) {
	issues := []IssueRow{
		// All similarities below weak threshold (sim < 0.3)
		{ID: "1", Repo: "demo/reporadar", Title: "r1", Distance: -0.2}, // sim 0.2
		{ID: "2", Repo: "demo/reporadar", Title: "r2", Distance: -0.1}, // sim 0.1
		{ID: "3", Repo: "demo/reporadar", Title: "r3", Distance: 0.05}, // sim -0.05
	}

	thr := Thresholds{
		Strong: 0.6,
		Weak:   0.3,
	}

	got := ScoreAndRank(issues, 10, thr)

	if len(got) != 0 {
		t.Fatalf("expected 0 results when all similarities are below weak threshold, got %d", len(got))
	}
}
