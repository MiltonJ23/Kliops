# Data Model

## PostgreSQL Schema

All tables are created by the migration `20260405103115-init_knowledge_base.sql`.

### appels_offres

Stores tender project metadata imported from the archive manifest.

| Column      | Type         | Constraints               |
|-------------|--------------|---------------------------|
| id          | UUID         | PRIMARY KEY, auto-generated|
| external_id | VARCHAR(100) | UNIQUE NOT NULL           |
| titre       | TEXT         | NOT NULL                  |
| client      | VARCHAR(255) | NOT NULL                  |
| status      | VARCHAR(50)  | NOT NULL                  |
| created_at  | TIMESTAMPTZ  | DEFAULT NOW()             |

### documents

Links uploaded PDF files to their parent project.

| Column         | Type        | Constraints                          |
|----------------|-------------|--------------------------------------|
| id             | UUID        | PRIMARY KEY, auto-generated          |
| appel_offre_id | UUID        | FK -> appels_offres(id) ON DELETE CASCADE |
| type           | VARCHAR(50) | CHECK (type IN ('DCE', 'MEMOIRE'))   |
| minio_path     | TEXT        | NOT NULL                             |
| created_at     | TIMESTAMPTZ | DEFAULT NOW()                        |

### processing_jobs

Tracks the async processing state for each project.

| Column         | Type        | Constraints                                    |
|----------------|-------------|------------------------------------------------|
| id             | UUID        | PRIMARY KEY, auto-generated                    |
| appel_offre_id | UUID        | FK -> appels_offres(id) ON DELETE CASCADE      |
| status         | VARCHAR(50) | CHECK (IN PENDING, PROCESSING, COMPLETED, FAILED)|
| error_message  | TEXT        |                                                |
| retry_count    | INT         | DEFAULT 0                                      |
| created_at     | TIMESTAMPTZ | DEFAULT NOW()                                  |
| updated_at     | TIMESTAMPTZ | DEFAULT NOW(), auto-updated via trigger        |

### reponses_historiques

Stores extracted requirement-response pairs with a link to the Qdrant vector.

| Column            | Type        | Constraints                          |
|-------------------|-------------|--------------------------------------|
| id                | UUID        | PRIMARY KEY, auto-generated          |
| appel_offre_id    | UUID        | FK -> appels_offres(id) ON DELETE CASCADE |
| exigence_technique| TEXT        | NOT NULL                             |
| reponse_apportee  | TEXT        | NOT NULL                             |
| qdrant_point_id   | UUID        | UNIQUE NOT NULL                      |
| created_at        | TIMESTAMPTZ | DEFAULT NOW()                        |

## Qdrant Vector Store

**Collection**: `memoire_technique`

| Field               | Type     | Description                          |
|---------------------|----------|--------------------------------------|
| vector              | float32[]| 1024-dimensional embedding           |
| ao_id               | string   | Project UUID                         |
| exigence_technique  | string   | Technical requirement text           |
| reponse_apportee    | string   | Response text from historical tender |
| gagne               | bool     | Whether the original tender was won  |
