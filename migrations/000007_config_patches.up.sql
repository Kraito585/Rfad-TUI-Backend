CREATE TABLE configs_patches (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    active BOOLEAN NOT NULL DEFAULT true,
    config JSONB NOT NULL
);