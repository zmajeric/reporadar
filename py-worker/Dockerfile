
FROM python:3.12-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt && python -m textblob.download_corpora

COPY . .

ENV DATABASE_URL=postgres://zmajeric:l0c4l@postgres:5432/rr

CMD ["python", "worker.py"]
