# Architecture

Kliops follows a hexagonal architecture (ports and adapters) to decouple domain logic from external infrastructure.

## Layer Overview

```
cmd/kliops-api/main.go          Entry point, dependency wiring, HTTP server
internal/core/domain/            Domain entities (AppelOffre, ReponseHistorique)
internal/core/ports/             Interface contracts (FileStorage, KnowledgeBase, etc.)
internal/core/services/          Business orchestration (ArchiveService, WorkerService, etc.)
internal/adapters/handlers/      HTTP handlers and middleware
internal/adapters/repositories/  PostgreSQL, MinIO, Qdrant implementations
internal/adapters/llm/           Gemma extractor, Ollama embedder
internal/adapters/queue/         RabbitMQ adapter
internal/adapters/parser/        PDF text extraction
internal/adapters/google_workspace/  Google Docs integration
```

## Port Interfaces

| Port               | File                        | Methods                                                   |
|--------------------|-----------------------------|-----------------------------------------------------------|
| FileStorage        | ports/storage.go            | Upload, DownloadStream                                    |
| KnowledgeBase      | ports/knowledge.go          | UpsertVector, SearchSimilar, DeleteByIDs                  |
| Embedder           | ports/knowledge.go          | CreateEmbedding                                           |
| IngestionRepository| ports/ingestion.go          | CreateProject, SaveDocument, CreateJob, UpdateJobStatus, GetDocumentPath |
| MessageQueue       | ports/ingestion.go          | PublishJob                                                |
| LLMExtractor       | ports/ingestion.go          | ExtractRequirementsAndAnswers                             |
| DocumentParser     | ports/ingestion.go          | FetchAndParse                                             |
| PricingStrategy    | ports/mercuriale.go         | GetPrice                                                  |
| DocumentGenerator  | ports/document_generator.go | GenerateFromStream, ShareWithUser                         |

## Adapter Implementations

| Adapter              | Implements       | External Dependency |
|----------------------|------------------|---------------------|
| MinioStorage         | FileStorage      | MinIO S3            |
| QdrantRepository     | KnowledgeBase    | Qdrant gRPC         |
| OllamaEmbedder       | Embedder         | Ollama HTTP API     |
| IngestionPostgres    | IngestionRepository | PostgreSQL       |
| RabbitMQAdapter      | MessageQueue     | RabbitMQ AMQP       |
| GemmaExtractor       | LLMExtractor     | Ollama + Gemma      |
| MinioPDFParser       | DocumentParser   | MinIO + ledongthuc/pdf |
| PricingPostgres      | PricingStrategy  | PostgreSQL          |
| PricingExcel         | PricingStrategy  | MinIO + excelize    |
| ERPPricing           | PricingStrategy  | External HTTP API   |
| WorkspaceAdapter     | DocumentGenerator| Google Workspace API|

## Design Decisions

- **Strategy Pattern** for pricing: three interchangeable sources (PostgreSQL, Excel, ERP).
- **Async Processing** via RabbitMQ with retry semantics (max 3 retries, exponential backoff, dead-letter queue).
- **Transactional Ingestion**: project creation, document uploads, and job creation wrapped in a single PostgreSQL transaction.
- **Streaming I/O**: PDF files streamed through temp files to avoid loading entire binaries in memory.
- **Semantic Retrieval**: DCE chunks are matched against MEMOIRE chunks via cosine similarity before being sent to the LLM.
