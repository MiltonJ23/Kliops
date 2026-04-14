
-- +migrate Up
CREATE TABLE appels_offres (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(100) UNIQUE NOT NULL,
    titre TEXT NOT NULL,
    client VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appel_offre_id UUID NOT NULL REFERENCES appels_offres(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('DCE', 'MEMOIRE')),
    minio_path TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_documents_appel_offre_id ON documents(appel_offre_id);

CREATE TABLE processing_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appel_offre_id UUID NOT NULL REFERENCES appels_offres(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')),
    error_message TEXT,
    retry_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_processing_jobs_appel_offre_id ON processing_jobs(appel_offre_id);
CREATE INDEX idx_processing_jobs_status ON processing_jobs(status);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +migrate StatementEnd

CREATE TRIGGER update_processing_jobs_updated_at BEFORE UPDATE ON processing_jobs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE reponses_historiques (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appel_offre_id UUID NOT NULL REFERENCES appels_offres(id) ON DELETE CASCADE,
    exigence_technique TEXT NOT NULL,
    reponse_apportee TEXT NOT NULL,
    qdrant_point_id UUID NOT NULL UNIQUE, -- Lien exact avec la DB Vectorielle
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_reponses_historiques_appel_offre_id ON reponses_historiques(appel_offre_id);

CREATE TABLE mercuriale (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code_article VARCHAR(100) UNIQUE,
    designation TEXT NOT NULL,
    unite VARCHAR(50),
    prix_unitaire NUMERIC(12,2) NOT NULL,
    source VARCHAR(100) DEFAULT 'postgres',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_mercuriale_code_article ON mercuriale(code_article);
-- +migrate Down
DROP TABLE mercuriale;
DROP TABLE reponses_historiques;
DROP TABLE processing_jobs;
DROP TABLE documents;
DROP TABLE appels_offres;
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;