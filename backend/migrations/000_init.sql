-- Migration: 000_init
-- Description: Database-level prerequisites. Run once before all other migrations.

-- Require pgcrypto for gen_random_uuid() on PG < 13.
-- PG 13+ has it built in via gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- trigram indexes for fuzzy search

-- Reusable trigger function to auto-update updated_at columns.
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
