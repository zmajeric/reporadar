import os
from typing import List

import psycopg2
import worker
from fastapi import FastAPI
from psycopg2.extras import register_default_json
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer

register_default_json(loads=lambda x: x)

DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgres://zmajeric:l0c4l@localhost:5432/rr",
)

app = FastAPI(title="RepoRadar Embedding API")


def get_connection():
    return psycopg2.connect(DATABASE_URL)


def load_model() -> SentenceTransformer:
    # Download model from Kaggle Models (cached locally)
    model = SentenceTransformer("sentence-transformers/all-MiniLM-L6-v2")
    return model


# Load once at startup
model: SentenceTransformer | None = None


@app.on_event("startup")
def on_startup():
    global model
    print("Loading embedding model from HuggingFace...")
    model = load_model()
    print("Model loaded.")


class EmbedRequest(BaseModel):
    text: str


class EmbedResponse(BaseModel):
    embedding: List[float]


@app.get("/health")
def health():
    # Optional: ping DB as well
    try:
        conn = get_connection()
        conn.close()
    except Exception as e:
        return {"status": "error", "error": str(e)}
    return {"status": "ok"}


@app.post("/embed", response_model=EmbedResponse)
def embed(req: EmbedRequest):
    if not req.text.strip():
        return EmbedResponse(embedding=[])
    emb = worker.compute_embedding(req.text, model)
    return EmbedResponse(embedding=emb)
