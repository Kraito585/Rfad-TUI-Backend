CREATE TABLE community_shader_presets (
    id UUID PRIMARY KEY,
    url TEXT NOT NULL,
    images TEXT[] NOT NULL DEFAULT '{}',
    
    -- 0: Light/Performance, 1: Medium/Quality, 2: Heavy/Extreme
    performance_impact SMALLINT NOT NULL CHECK (performance_impact IN (0, 1, 2)),
    
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);