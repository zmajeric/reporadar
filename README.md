
# RepoRadar – GitHub Issue Insights

RepoRadar is a small system that:
- Ingests GitHub issues (or mock JSON issues)
- Stores them in Postgres + pgvector
- Uses a Python worker to compute embeddings and keywords
- Exposes endpoints to search and analyze issues

Folder structure:
- `go-ingest/` – Go HTTP server skeleton for ingest & read APIs
- `py-worker/` – Python worker skeleton with Kaggle Models + sentence-transformers
- `sql/` – SQL to set up Postgres + pgvector schema
- `data/` – Mock issues JSON file
- `docs/` – Architecture notes placeholder

## Quick start

1. Start Postgres with pgvector:
   ```bash
   docker compose up -d
   ```

2. Run the Go service:
   ```bash
   cd go-ingest
   go run ./...
   ```

3. Run the Python worker (after installing deps):
   ```bash
   cd py-worker
   pip install -r requirements.txt
   python worker.py
   ```

