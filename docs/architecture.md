
# RepoRadar Architecture (Draft)

Components:

- **Go ingest service (`go-ingest/`)**
  - HTTP API
  - Ingests issues (mock or GitHub)
  - Stores raw issues in Postgres

- **Python worker (`py-worker/`)**
  - Fetches issues without embeddings
  - Uses Kaggle Models + sentence-transformers to compute embeddings
  - Extracts simple keywords
  - Updates Postgres rows

- **Postgres + pgvector**
  - Stores issues and their embeddings
  - Supports similarity search and insights queries
