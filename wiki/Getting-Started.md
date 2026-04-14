# Getting Started

## Prerequisites

- Go 1.25+
- Docker and Docker Compose
- Ollama with `gemma` and `mxbai-embed-large` models pulled
- (Optional) `sql-migrate` for database migrations

## Clone and Build

```
git clone https://github.com/MiltonJ23/Kliops.git
cd Kliops
make build
```

## Start Infrastructure

Create a `.env` file at the repository root:

```
POSTGRES_USER=kliops
POSTGRES_PASSWORD=kliops
POSTGRES_DB=kliops
DB_DSN=postgres://kliops:kliops@localhost:5432/kliops
MINIO_ENDPOINT=localhost:9000
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_USE_SSL=false
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_URI=amqp://guest:guest@localhost:5672/
API_KEY_SECRET=changeme
```

Start the backing services:

```
make docker-up
```

## Run Migrations

```
sql-migrate up
```

## Start the API

```
make run
```

The server listens on port `8070`. Verify:

```
curl http://localhost:8070/health
```

## Pull Ollama Models

```
ollama pull gemma
ollama pull mxbai-embed-large
```

## Run Tests

```
make test
```

## Docker Image

Build and run the API as a container:

```
docker build -t kliops-api .
docker run --rm -p 8070:8070 --env-file .env kliops-api
```
