CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 会話要約テーブル（ベクトル検索用）
CREATE TABLE conversation_summaries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    vector vector(1536) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_user_timerange UNIQUE (user_id, start_time, end_time)
);

-- ベクトル類似度検索用のインデックス
CREATE INDEX ON conversation_summaries USING ivfflat (vector vector_cosine_ops)
WITH (lists = 100);

-- 時間範囲検索用のインデックス
CREATE INDEX idx_conversation_summaries_time_range 
ON conversation_summaries (user_id, start_time, end_time);