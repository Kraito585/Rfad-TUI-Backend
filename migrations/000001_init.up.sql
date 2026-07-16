CREATE TABLE IF NOT EXISTS sync_worker_history (
    event_id UUID NOT NULL, 
    pod_id VARCHAR(255), 
    event_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_history_synced_at ON sync_worker_history(created_at);