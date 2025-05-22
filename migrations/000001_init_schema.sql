-- Migration: init_schema
-- Created at: 2024-03-20

-- Up Migration
-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create polls table
CREATE TABLE polls (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create poll_options table
CREATE TABLE poll_options (
    id UUID PRIMARY KEY,
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    option_text TEXT NOT NULL,
    option_index INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(poll_id, option_index)
);

-- Create poll_tags table
CREATE TABLE poll_tags (
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    tag VARCHAR(50) NOT NULL,
    PRIMARY KEY (poll_id, tag)
);

-- Create votes table
CREATE TABLE votes (
    id UUID PRIMARY KEY,
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    option_id UUID NOT NULL REFERENCES poll_options(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(poll_id, user_id)
);

-- Create skips table
CREATE TABLE skips (
    id UUID PRIMARY KEY,
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(poll_id, user_id)
);

-- Create user_daily_votes table
CREATE TABLE user_daily_votes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vote_date DATE NOT NULL,
    vote_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(user_id, vote_date)
);

-- Create indexes
CREATE INDEX idx_polls_created_at ON polls(created_at);
CREATE INDEX idx_poll_options_poll_id ON poll_options(poll_id);
CREATE INDEX idx_poll_tags_poll_id ON poll_tags(poll_id);
CREATE INDEX idx_poll_tags_tag ON poll_tags(tag);
CREATE INDEX idx_votes_poll_id ON votes(poll_id);
CREATE INDEX idx_votes_user_id ON votes(user_id);
CREATE INDEX idx_skips_poll_id ON skips(poll_id);
CREATE INDEX idx_skips_user_id ON skips(user_id);
CREATE INDEX idx_user_daily_votes_user_date ON user_daily_votes(user_id, vote_date);

-- Down Migration
-- Drop indexes
DROP INDEX IF EXISTS idx_user_daily_votes_user_date;
DROP INDEX IF EXISTS idx_skips_user_id;
DROP INDEX IF EXISTS idx_skips_poll_id;
DROP INDEX IF EXISTS idx_votes_user_id;
DROP INDEX IF EXISTS idx_votes_poll_id;
DROP INDEX IF EXISTS idx_poll_tags_tag;
DROP INDEX IF EXISTS idx_poll_tags_poll_id;
DROP INDEX IF EXISTS idx_poll_options_poll_id;
DROP INDEX IF EXISTS idx_polls_created_at;

-- Drop tables
DROP TABLE IF EXISTS user_daily_votes;
DROP TABLE IF EXISTS skips;
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS poll_tags;
DROP TABLE IF EXISTS poll_options;
DROP TABLE IF EXISTS polls;
DROP TABLE IF EXISTS users; 