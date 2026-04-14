# Ingestion Pipeline

The ingestion pipeline transforms raw tender documents into a structured, searchable knowledge base.

## Overview

```
ZIP Upload -> Extract -> PostgreSQL Transaction -> MinIO Upload -> RabbitMQ Job -> Worker Processing
```

## Step-by-Step Flow

### 1. Archive Upload

`POST /api/v1/ingest/archive` receives a ZIP file containing:
- `manifest.csv` with columns: `id_projet`, `titre`, `client`, `statut`, `fichier_dce`, `fichier_memoire`
- PDF files referenced by the manifest

### 2. Synchronous Processing (ArchiveService)

For each row in the manifest:

1. Begin PostgreSQL transaction
2. Create project entry in `appels_offres` (skip if `external_id` already exists)
3. Extract referenced PDF files from the ZIP
4. Upload each PDF to MinIO bucket `dce-archive` under path `{projectID}/{filename}`
5. Insert document records in the `documents` table
6. Create a processing job in `processing_jobs` with status `PENDING`
7. Commit transaction
8. Publish job message to RabbitMQ exchange `kliops_ingestion`

### 3. Asynchronous Processing (WorkerService)

The worker consumes from the `ingestion_jobs` queue:

1. Set job status to `PROCESSING`
2. Retrieve document paths from PostgreSQL
3. For each document (DCE and MEMOIRE):
   - Download PDF from MinIO via streaming
   - Extract text using `ledongthuc/pdf`
   - Split text into 2500-character chunks
   - Generate embeddings for each chunk using `mxbai-embed-large`
4. For each DCE chunk:
   - Embed the chunk
   - Find top-3 semantically similar MEMOIRE chunks (cosine similarity > 0.40)
   - Send the DCE chunk and matched MEMOIRE context to Gemma
   - Parse JSON response containing `(exigence, reponse)` pairs
5. For each extracted pair:
   - Generate embedding for the `exigence_technique`
   - Upsert vector into Qdrant collection `memoire_technique`
   - Insert record into `reponses_historiques` table
6. Set job status to `COMPLETED`

### 4. Error Handling

- Failed jobs are retried up to 3 times
- Backoff delay: `retry_count * 5 seconds`
- After 3 failures, the message is routed to the dead-letter queue (`ingestion_dlq`)
- On vector ingestion failure, all Qdrant points for the current job are rolled back

## Qdrant Collection Schema

- **Collection**: `memoire_technique`
- **Vector dimension**: 1024
- **Payload fields**:
  - `ao_id` (string): project UUID
  - `exigence_technique` (string): extracted requirement
  - `reponse_apportee` (string): historical response
  - `gagne` (bool): whether the tender was won
