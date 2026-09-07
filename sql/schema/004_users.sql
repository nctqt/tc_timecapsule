-- +goose Up

CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    hashed_password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE watch_history (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    watched_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT TRUE,
    
    CONSTRAINT unique_user_video UNIQUE (user_id, video_id)
);

CREATE INDEX idx_watch_history_user_id ON watch_history(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_watch_history_user_id;
DROP TABLE IF EXISTS watch_history;
DROP TABLE IF EXISTS users;