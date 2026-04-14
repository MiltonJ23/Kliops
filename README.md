# Kliops

Kliops is an AI-driven tender response automation platform for the BTP (civil engineering) sector. It ingests Dossiers de Consultation des Entreprises (DCE), builds a searchable vector knowledge base from past responses, and drives a multi-step LLM agent — backed by a local Ollama/Gemma instance — that retrieves relevant precedents, prices bill-of-quantities line items, and generates a structured technical memorandum (Memoire Technique) via Google Docs.

The system is built around hexagonal architecture: the domain and service layer are fully isolated from infrastructure adapters (Postgres, MinIO, RabbitMQ, Qdrant, Ollama, Google Workspace). Each adapter is replaceable by implementing the corresponding port interface.

---

## Architecture

```
                      HTTP (Go net/http)
                             |
                    APIKeyMiddleware
                             |
         ┌───────────────────┼──────────────────────┐
         |                   |                       |
   IngestionHandler    AgentHandler           GatewayHandler
         |                   |                       |
   ArchiveService      AgentService            PricingService
         |              (orchestrator)              |
   IngestionRepo     KliopsOrchestrator        strategies
   (Postgres)        (multi-agent loop,         ├── postgres
         |            tool-calling via          ├── excel
   RabbitMQ           Ollama /v1 OpenAI API)    └── erp (optional)
   (job queue)              |
         |          ┌───────┼────────────┐
   WorkerService    |       |            |
   (async consumer) |       |            |
         |    KnowledgeService  PricingService  DocumentService
   GemmaExtractor   (Qdrant)    (see above)   (Google Docs)
   (embeddings +          |
    text extraction)  QdrantRepo
         |             (vectors)
   MinioPDFParser
   (MinIO → PDF text)
```

Ingest path:
```
POST /api/v1/ingest/archive  →  ArchiveService.ProcessZipArchive
  → unzips DCE + MEMOIRE PDFs
  → uploads to MinIO (dce-archive bucket)
  → writes appel_offre + documents rows in Postgres
  → publishes job to RabbitMQ

RabbitMQ consumer (WorkerService)
  → downloads PDF from MinIO
  → extracts text (GemmaExtractor)
  → embeds chunks (OllamaEmbedder)
  → stores vectors in Qdrant with metadata
```

Agent path:
```
POST /api/v1/agent/ask
  → AgentService.ProcessTender
  → builds system + user prompt
  → KliopsOrchestrator.Run (tool-calling loop, max 12 iterations)
      tool: search_knowledge   → KnowledgeService → Qdrant kNN
      tool: get_price          → PricingService   → configured strategy
      tool: generate_document  → DocumentService  → Google Drive/Docs
  → returns document URL and status message
```

---

## API Reference

All routes under `/api/v1/` require the header `X-API-KEY: <API_KEY_SECRET>`.

### System probes

| Method | Path      | Auth | Description                               |
|--------|-----------|------|-------------------------------------------|
| GET    | /         | none | Service identity and uptime               |
| GET    | /livez    | none | Liveness: always 200 while process is up  |
| GET    | /health   | none | Alias for /livez                          |
| GET    | /readyz   | none | Readiness: checks all five subsystems     |
| GET    | /version  | none | Build version string                      |

`GET /readyz` response:
```json
{
  "status": "READY",
  "checks": [
    {"name": "postgres", "ok": true},
    {"name": "minio",    "ok": true},
    {"name": "rabbitmq", "ok": true},
    {"name": "qdrant",   "ok": true},
    {"name": "ollama",   "ok": true}
  ]
}
```

### Ingestion

#### `POST /api/v1/ingest/archive`

Accepts a multipart form with a ZIP archive field named `archive`. The ZIP must contain a `manifest.json` describing the project and pointing to DCE/MEMOIRE PDF files within the archive.

Manifest format (`manifest.json`):
```json
{
  "external_id": "AO-2026-001",
  "titre": "Construction de la salle des fetes",
  "client": "Mairie de Yaonde",
  "fichier_dce": "dce.pdf",
  "fichier_mem": "memoire.pdf"
}
```

Returns `202 Accepted` with the created job ID.

#### `POST /api/v1/ingest/mercuriale`

Multipart form, field `excel_file`. Uploads an Excel price list (`.xlsx`) to MinIO under `kliops-config/mercuriale_current.xlsx`. Enables the `excel` pricing strategy for subsequent agent runs.

#### `POST /api/v1/ingest/template`

Multipart form, field `template`. Uploads a `.docx` template to MinIO under `kliops-config/template_charte.docx`. Used by the document generation step of the agent pipeline.

#### `POST /api/v1/upload`

General-purpose file upload. Multipart form, field `document`. Stores in MinIO bucket `dce-entrants`. Returns the MinIO object path.

### Agent

#### `POST /api/v1/agent/ask`

Runs the full agentic pipeline for a single tender.

Request body:
```json
{
  "dce_content":  "<full extracted text of the DCE document>",
  "project_name": "Construction salle des fetes Yaounde",
  "target_email": "chef.projet@entreprise.com",
  "client_name":  "Mairie de Yaounde"
}
```

`dce_content` must be between 50 and 500,000 characters.

Response (200 OK):
```json
{
  "message":     "Document genere: https://docs.google.com/document/d/.../edit",
  "duration_ms": 42380
}
```

Error codes:
- `400` — malformed or unparseable JSON
- `413` — request body exceeds 10 MB
- `422` — domain validation failure (short content, missing fields)
- `504` — pipeline did not complete within 10 minutes
- `500` — unexpected internal failure

### Pricing

#### `GET /api/v1/price?source=<strategy>&code=<code_article>`

Query a single price from a registered pricing strategy.

| Parameter | Values                         |
|-----------|--------------------------------|
| `source`  | `postgres`, `excel`, `erp`     |
| `code`    | article code (e.g. `BET001`)   |

Response (200 OK):
```json
{
  "source":       "postgres",
  "code_article": "BET001",
  "prix":         185.50
}
```

- `400` — missing or empty parameters
- `404` — code not found in the specified source
- `500` — strategy not registered or infrastructure failure

---

## Configuration

All configuration is via environment variables. Populate a `.env` file at the repository root for local development; the application loads it via `godotenv` at startup.

| Variable               | Required | Default            | Description                                              |
|------------------------|----------|--------------------|----------------------------------------------------------|
| `DATABASE_URL`         | yes      | —                  | Postgres DSN: `postgres://user:pass@host:5432/db?sslmode=disable` |
| `MINIO_ENDPOINT`       | yes      | —                  | MinIO host:port (no scheme)                              |
| `MINIO_ROOT_USER`      | yes      | —                  | MinIO access key                                         |
| `MINIO_ROOT_PASSWORD`  | yes      | —                  | MinIO secret key                                         |
| `MINIO_USE_SSL`        | no       | `false`            | Set `true` for TLS MinIO endpoints                       |
| `RABBITMQ_URL`         | yes      | —                  | AMQP URI: `amqp://user:pass@host:5672/`. URL-encode special chars in password. |
| `QDRANT_ADDR`          | yes      | —                  | Qdrant gRPC address: `host:6334`                         |
| `QDRANT_COLLECTION`    | no       | `btp_knowledge`    | Qdrant collection name                                   |
| `OLLAMA_BASE_URL`      | no       | `http://localhost:11434` | Base URL for Ollama API                             |
| `OLLAMA_CHAT_MODEL`    | no       | `gemma4:e4b`       | Chat/generation model tag                                |
| `OLLAMA_EMBED_MODEL`   | no       | `mxbai-embed-large`| Embedding model tag                                      |
| `GOOGLE_CREDENTIALS_FILE` | no    | `./credentials.json` | Path to Google service account JSON key file           |
| `PRICING_EXCEL_PATH`   | no       | `./dummy_prices.xlsx` | Path to Excel price list. If absent, `excel` strategy is disabled. |
| `ERP_BASE_URL`         | no       | —                  | Base URL for ERP pricing API. If unset, `erp` strategy is disabled. |
| `API_KEY_SECRET`       | yes      | —                  | Pre-shared key required in `X-API-KEY` header            |
| `APP_ADDR`             | no       | `:8070`            | TCP address to bind (`host:port` or `:port`)             |

The application refuses to start if any required variable is missing. It also validates connectivity to Postgres, MinIO, RabbitMQ, and Qdrant during initialization.

---

## Database Schema

Run `deployments/migrations/20260405103115-init_knowledge_base.sql` against your Postgres instance to create all required tables. The migration uses [sql-migrate](https://github.com/rubenv/sql-migrate) format (`+migrate Up`/`+migrate Down` directives); use `sql-migrate up` or apply the Up section manually with `psql`.

Tables:
- `appels_offres` — tender records
- `documents` — references to MinIO objects (DCE and MEMOIRE)
- `processing_jobs` — async job state machine (PENDING → PROCESSING → COMPLETED/FAILED)
- `reponses_historiques` — vector knowledge base metadata
- `mercuriale` — unit price reference table

---

## Development

### Prerequisites

- Go 1.22 or later
- Docker and Docker Compose
- An Ollama installation with `gemma4:e4b` and `mxbai-embed-large` pulled:
  ```
  ollama pull gemma4:e4b
  ollama pull mxbai-embed-large
  ```
- A Google Cloud service account with Drive and Docs API enabled. Place the JSON key at `./credentials.json`.

### Starting local infrastructure

```sh
docker compose --env-file .env -f deployments/docker-compose.yml up -d
```

This brings up Postgres 15, MinIO, RabbitMQ 4, and Qdrant. Wait for all health checks to pass before running the application.

### Applying migrations

```sh
# requires sql-migrate: go install github.com/rubenv/sql-migrate/...@latest
DB_DSN=$DATABASE_URL sql-migrate up -env development
# or manually:
docker exec -i kliops-postgres psql -U kliops -d kliops < deployments/migrations/20260405103115-init_knowledge_base.sql
```

### Building and running

```sh
make build    # go build -o bin/kliops-api cmd/kliops-api/main.go
make run      # build + ./bin/kliops-api
```

The binary reads `.env` automatically via `godotenv`.

### Running with live reload (optional)

```sh
go install github.com/air-verse/air@latest
air
```

---

## Testing

### Unit tests

```sh
make test
# or with coverage:
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Integration smoke tests

With infrastructure running and `./bin/kliops-api` listening on `:8070`:

```sh
# Liveness
curl http://localhost:8070/livez

# Readiness (all five subsystems)
curl http://localhost:8070/readyz

# Pricing lookup (requires at least one mercuriale row)
curl -H "X-API-KEY: $API_KEY_SECRET" \
     "http://localhost:8070/api/v1/price?source=postgres&code=BET001"

# Agent pipeline (minimum 50-char DCE content required)
curl -X POST http://localhost:8070/api/v1/agent/ask \
     -H "Content-Type: application/json" \
     -H "X-API-KEY: $API_KEY_SECRET" \
     -d '{
       "dce_content":  "Fourniture et pose de beton arme 30MPa pour semelles filantes. Surface estimee a 250m2. Dosage ciment CPJ 45. Armatures acier HA 500. Fouilles mecaniques incluses.",
       "project_name": "Immeuble R+4 Bastos Yaounde",
       "target_email": "ingenieur@entreprise.cm",
       "client_name":  "Societe Immobiliere du Centre"
     }'
```

### Test coverage targets

| Package                              | Focus                                    |
|--------------------------------------|------------------------------------------|
| `internal/core/services`             | AgentService, PricingService validation  |
| `internal/adapters/handlers`         | HTTP status codes, JSON serialization    |
| `cmd/kliops-api`                     | Route registration, StripPrefix behavior |

---

## Deployment

### Docker

```sh
docker build --build-arg VERSION=$(git describe --tags --always) -t kliops-api .
docker run --env-file .env -p 8070:8070 kliops-api
```

The image uses a non-root user (`kliops`). The binary is statically linked (`CGO_ENABLED=0`); no external library dependencies at runtime.

### Kubernetes

Manifests are in `deployments/k8s/kliops.yaml`. They define a Namespace, ConfigMap, Secret, Deployment (2 replicas by default), ClusterIP Service, and Ingress.

Before applying, substitute the three placeholder tokens:

```sh
sed \
  -e "s|IMAGE_REPOSITORY_PLACEHOLDER|ghcr.io/miltonj23/kliops|g" \
  -e "s|IMAGE_TAG_PLACEHOLDER|v1.0.0|g" \
  -e "s|NAMESPACE_PLACEHOLDER|kliops|g" \
  deployments/k8s/kliops.yaml | kubectl apply -f -
```

The Deployment configures:
- Liveness probe: `GET /livez` (initial delay 10s, period 15s)
- Readiness probe: `GET /readyz` (initial delay 15s, period 10s)
- Resource requests/limits: 250m/500m CPU, 128Mi/256Mi memory
- `GOOGLE_CREDENTIALS_FILE` must be mounted as a secret volume in production

### CI/CD workflows

| Workflow                      | Trigger                | Action                                       |
|-------------------------------|------------------------|----------------------------------------------|
| `.github/workflows/ci.yml`    | push to main, all PRs  | vet, race tests with coverage, build binary  |
| `.github/workflows/release.yml` | push of `v*` tag     | multi-arch Docker build, push to GHCR        |
| `.github/workflows/deploy.yml`  | manual (`workflow_dispatch`) | render k8s manifest, `kubectl apply` |

For the deploy workflow, set repository secrets `KUBE_CONFIG_B64` (base64-encoded kubeconfig) in your GitHub repository settings.

---

## Project Structure

```
.
├── cmd/
│   └── kliops-api/           Application entrypoint
│       └── main.go           Bootstrap: config, wiring, HTTP server, readiness
├── internal/
│   ├── core/
│   │   ├── domain/           Value objects and domain types
│   │   ├── ports/            Interface definitions (FileStorage, PricingStrategy, DocumentGenerator, ...)
│   │   └── services/         Application services (AgentService, ArchiveService, WorkerService, ...)
│   └── adapters/
│       ├── agent/            KliopsOrchestrator: Ollama tool-calling loop
│       ├── google_workspace/ WorkspaceAdapter: Drive upload, Docs batch update
│       ├── handlers/         HTTP handlers and middleware
│       ├── llm/              GemmaExtractor (text extraction), OllamaEmbedder
│       ├── parser/           MinioPDFParser: downloads PDF from MinIO, extracts text
│       ├── queue/            RabbitMQAdapter: publisher + consumer with DLQ
│       └── repositories/     Postgres, MinIO, Qdrant, pricing adapters
├── deployments/
│   ├── docker-compose.yml    Local infrastructure (Postgres, MinIO, RabbitMQ, Qdrant)
│   ├── migrations/           SQL migration files (sql-migrate format)
│   └── k8s/
│       └── kliops.yaml       Kubernetes manifests
├── .github/workflows/        CI, release, and deploy pipelines
├── Dockerfile                Multi-stage production image
└── Makefile                  build, run, test, docker-up, docker-down targets
```

---

## License

See [LICENSE](LICENSE).
