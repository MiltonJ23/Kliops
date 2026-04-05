
-- +migrate Up
CREATE TABLE appels_offres (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(100) UNIQUE NOT NULL, -- ID issu du CSV
    titre TEXT NOT NULL,
    client VARCHAR(255) NOT NULL,
    statut VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appel_offre_id UUID NOT NULL REFERENCES appels_offres(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('DCE', 'MEMOIRE')),
    minio_path TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE processing_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appel_offre_id UUID NOT NULL REFERENCES appels_offres(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')),
    error_message TEXT,
    retry_count INT DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE reponses_historiques (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appel_offre_id UUID NOT NULL REFERENCES appels_offres(id) ON DELETE CASCADE,
    exigence_technique TEXT NOT NULL,
    reponse_apportee TEXT NOT NULL,
    qdrant_point_id UUID NOT NULL UNIQUE, -- Lien exact avec la DB Vectorielle
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
-- +migrate Down
DROP TABLE reponses_historiques;
DROP TABLE processing_jobs;
DROP TABLE documents;
DROP TABLE appels_offres;