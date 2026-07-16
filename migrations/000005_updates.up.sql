CREATE TABLE IF NOT EXISTS app_updates (
    id UUID PRIMARY KEY,                   -- UUIDv7 (внутренняя версия / идентификатор)
    remote_version VARCHAR(50) NOT NULL,   -- Версия из Google Docs (dd.mm) для проверок
    url VARCHAR(255) NOT NULL,             -- Ссылка на скачивание (или путь в S3)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индекс для быстрого поиска последней версии (UUIDv7 сортируется хронологически)
CREATE INDEX idx_app_updates_id ON app_updates(id DESC);