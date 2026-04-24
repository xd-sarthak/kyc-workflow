-- KYC Workflow Service — Initial Schema
-- Run against PostgreSQL 14+

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT UNIQUE NOT NULL,
    password   TEXT NOT NULL,
    role       TEXT NOT NULL CHECK (role IN ('merchant', 'reviewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- KYC submissions table
CREATE TABLE IF NOT EXISTS kyc_submissions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state             TEXT NOT NULL DEFAULT 'draft',
    personal_details  JSONB,
    business_details  JSONB,
    reviewer_note     TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_kyc_submissions_merchant_id ON kyc_submissions(merchant_id);
CREATE INDEX IF NOT EXISTS idx_kyc_submissions_state ON kyc_submissions(state);

-- Documents table
CREATE TABLE IF NOT EXISTS documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id   UUID NOT NULL REFERENCES kyc_submissions(id) ON DELETE CASCADE,
    file_type       TEXT NOT NULL CHECK (file_type IN ('pan', 'aadhaar', 'bank_statement')),
    storage_key     TEXT NOT NULL,
    storage_backend TEXT NOT NULL CHECK (storage_backend IN ('local', 's3')),
    original_name   TEXT NOT NULL,
    mime_type       TEXT NOT NULL,
    size_bytes      BIGINT NOT NULL,
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_documents_submission_id ON documents(submission_id);

-- Unique constraint: one document per type per submission
CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_submission_file_type
    ON documents(submission_id, file_type);

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL,
    payload     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_merchant_id ON notifications(merchant_id);
