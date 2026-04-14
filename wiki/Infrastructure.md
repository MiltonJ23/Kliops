# Infrastructure

Kliops relies on five external services, all defined in `deployments/docker-compose.yml`.

## Services

| Service        | Image                   | Purpose                                    | Ports        |
|----------------|-------------------------|--------------------------------------------|--------------|
| PostgreSQL     | postgres:15-alpine      | Relational store for projects, jobs, prices| 5432         |
| MinIO          | minio/minio:latest      | S3-compatible object storage for PDFs      | 9000, 9001   |
| RabbitMQ       | rabbitmq:4-management   | Async job queue with management UI         | 5672, 15672  |
| Qdrant         | qdrant/qdrant:latest    | Vector database for semantic search        | 6333, 6334   |
| Ollama         | (external)              | LLM runtime hosting Gemma and embedder     | 11434        |

## AI Models

| Model              | Role                   | API Endpoint         | Vector Dimension |
|--------------------|------------------------|----------------------|------------------|
| Gemma              | Requirement extraction | POST /api/generate   | N/A              |
| mxbai-embed-large  | Text embeddings        | POST /api/embeddings | 1024             |

Both models are served through Ollama's HTTP interface on port 11434.

## Environment Variables

```
DB_DSN=postgres://user:password@localhost:5432/kliops
MINIO_ENDPOINT=localhost:9000
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_USE_SSL=false
RABBITMQ_URI=amqp://guest:guest@localhost:5672/
API_KEY_SECRET=<your-api-key>
```

## Storage Buckets

| Bucket         | Purpose                                      |
|----------------|----------------------------------------------|
| dce-entrants   | Uploaded DCE documents via /api/v1/upload     |
| dce-archive    | Extracted PDFs from ZIP archives              |
| kliops-config  | Mercuriale XLSX and DOCX template             |

## RabbitMQ Topology

| Exchange           | Queue            | Routing Key      | Purpose          |
|--------------------|------------------|------------------|------------------|
| kliops_ingestion   | ingestion_jobs   | ingestion.new    | New jobs         |
| kliops_dlx         | ingestion_dlq    | dlq              | Failed jobs (DLQ)|

Messages are retried up to 3 times with exponential backoff (`retry * 5s`) before being routed to the dead-letter queue.

## Database Migrations

Migrations live in `deployments/migrations/` and are managed via `sql-migrate` with `dbconfig.yml`.

Apply migrations:
```
sql-migrate up
```
