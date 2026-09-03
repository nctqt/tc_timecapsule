-- +goose Up
CREATE TABLE cases (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE milestones (
    id UUID PRIMARY KEY,
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    event_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_milestones_case_id ON milestones(case_id);

CREATE TABLE videos (
    id UUID PRIMARY KEY,
    milestone_id UUID REFERENCES milestones(id) ON DELETE SET NULL,
    youtube_video_id VARCHAR(50) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    channel_name VARCHAR(255) NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    category VARCHAR(50) NOT NULL -- 'news', 'speculation', 'podcast'
);

CREATE INDEX idx_videos_milestone_id ON videos(milestone_id);

-- +goose Down
DROP INDEX IF EXISTS idx_videos_milestone_id;
DROP TABLE IF EXISTS videos;
DROP INDEX IF EXISTS idx_milestones_case_id;
DROP TABLE IF EXISTS milestones;
DROP TABLE IF EXISTS cases;