import os
from typing import List

import psycopg2
from psycopg2.extras import register_default_json
from sentence_transformers import SentenceTransformer
from textblob import TextBlob

register_default_json(loads=lambda x: x)

DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgres://zmajeric:l0c4l@localhost:5432/rr",
)


def get_connection():
    return psycopg2.connect(DATABASE_URL)


def load_model() -> SentenceTransformer:
    # Download model from Kaggle Models (cached locally)
    model = SentenceTransformer("sentence-transformers/all-MiniLM-L6-v2")
    return model


def extract_keywords(text: str, max_keywords: int = 5) -> List[str]:
    blob = TextBlob(text)
    phrases = [p.lower() for p in blob.noun_phrases]
    words = [w.lower() for w in blob.words if len(w) > 3]
    seen = set()
    keywords: List[str] = []
    for w in phrases + words:
        if w not in seen:
            seen.add(w)
            keywords.append(w)
        if len(keywords) >= max_keywords:
            break
    return keywords


def fetch_issues_without_embeddings(conn):
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT id, title, COALESCE(body, ''), repo
            FROM issues
            WHERE embedding IS NULL
            ORDER BY created_at ASC
            LIMIT 50;
            """
        )
        return cur.fetchall()


def update_issue_embedding(conn, issue_id: str, embedding: list[float], keywords):
    with conn.cursor() as cur:
        cur.execute(
            """
            UPDATE issues
            SET embedding = %s::vector,
                keywords = %s
            WHERE id = %s;
            """,
            (embedding, keywords, issue_id),
        )
    conn.commit()

def compute_embedding(text: str, model: SentenceTransformer) -> list[float]:
    emb = model.encode(
        text,
        convert_to_numpy=True,
        normalize_embeddings=True,
    )
    return emb.tolist()

def main():
    print("Connecting to Postgres...")
    conn = get_connection()
    print("Loading embedding model from Hugging Face...")
    model = load_model()
    print("Ready. Processing issues without embeddings...")

    issues = fetch_issues_without_embeddings(conn)
    print(f"Found {len(issues)} issue(s) to process")

    for issue_id, title, body, repo in issues:
        text = f"{title}\n\n{body}"
        computed_embedding = compute_embedding(text, model)
        keywords = extract_keywords(text)
        print(f"Updating issue {issue_id} ({repo}) with {len(keywords)} keywords")
        update_issue_embedding(conn, issue_id, computed_embedding, keywords)

    conn.close()
    print("Done.")


if __name__ == "__main__":
    main()
