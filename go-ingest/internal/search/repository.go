package search

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmajeric/reporadar-go-ingest/utils"
)

type PgRepository struct {
	db *pgxpool.Pool
}

func NewPgRepository(db *pgxpool.Pool) *PgRepository {
	return &PgRepository{db: db}
}

func (pgr *PgRepository) SearchByVector(ctx context.Context, repo string, vector []float32, limit int) ([]IssueRow, error) {
	vectorLiteral := utils.EmbeddingToVectorLiteral(vector)

	const qSQL = `
		SELECT id, repo, title, body, embedding <=> $1::vector AS distance
		FROM issues
		WHERE repo = $2
		ORDER BY embedding <=> $1::vector
		LIMIT $3;
	`

	sqlStartTime := time.Now()
	rows, err := pgr.db.Query(ctx, qSQL, vectorLiteral, repo, limit)
	if err != nil {
		return nil, err
	}
	sqlProcTime := time.Since(sqlStartTime)
	log.Printf("searchByVector sql time: %v", sqlProcTime)

	var results []IssueRow
	for rows.Next() {
		var r IssueRow
		if err := rows.Scan(&r.ID, &r.Repo, &r.Title, &r.Body, &r.Distance); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}
